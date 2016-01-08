// Package bitcoin handles the bitcoin related parts of degdb.
package bitcoin

import (
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/btcsuite/btcrpcclient"
	"github.com/btcsuite/btcutil"
)

//go:generate go get -v -u github.com/btcsuite/btcwallet
//go:generate go get -v -u github.com/btcsuite/btcd

var (
	RPCUser = "degdbuser"

	TestNet = true
)

func launchProc(name string, args ...string) {
	cmd := exec.Command(name, args...)
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	defer cmd.Process.Kill()
	if err := cmd.Start(); err != nil {
		log.Println(err)
	}
	procChannel <- cmd
	if err := cmd.Wait(); err != nil {
		log.Println(err)
	}
}

var launchOnce sync.Once
var launchWG sync.WaitGroup
var client *btcrpcclient.Client
var procChannel chan *exec.Cmd
var procKillChannel chan bool

var procs []*exec.Cmd

func procHandler() {
	procChannel = make(chan *exec.Cmd, 1)
	procKillChannel = make(chan bool, 1)
	for {
		select {
		case proc := <-procChannel:
			procs = append(procs, proc)
		case <-procKillChannel:
			for _, proc := range procs {
				proc.Process.Kill()
			}
			procs = nil
		}
	}
}

// Kill kills all processes launched by bitcoin.
func Kill() {
	procKillChannel <- true
}

func init() {
	launchWG.Add(1)
	go procHandler()
}

func NewClient() (*btcrpcclient.Client, error) {
	var cerr error
	launchOnce.Do(func() {
		defer launchWG.Done()
		password := randString(40)
		go launchProc("btcd", "--testnet", "-u", RPCUser, "-P", password)
		go launchProc("btcwallet", "-u", RPCUser, "-P", password)

		time.Sleep(1 * time.Second)

		btcdHomeDir := btcutil.AppDataDir("btcd", false)
		certs, err := ioutil.ReadFile(filepath.Join(btcdHomeDir, "rpc.cert"))
		if err != nil {
			log.Fatal(err)
		}
		connCfg := &btcrpcclient.ConnConfig{
			Host:         "localhost:18334",
			Endpoint:     "ws",
			User:         RPCUser,
			Pass:         password,
			Certificates: certs,
		}
		_ = connCfg
		time.Sleep(2 * time.Second)
		client, cerr = btcrpcclient.New(connCfg, nil) // handlers)
		if cerr != nil {
			return
		}
	})
	launchWG.Wait()
	return client, cerr
}
