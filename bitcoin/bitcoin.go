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

	launchOnce  sync.Once
	launchWG    sync.WaitGroup
	client      *btcrpcclient.Client
	procChannel = make(chan *exec.Cmd, 1)

	procs     []*exec.Cmd
	procsLock sync.Mutex
)

func launchProc(name string, args ...string) {
	cmd := exec.Command(name, args...)
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	defer func() {
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
	}()
	if err := cmd.Start(); err != nil {
		log.Println(err)
		return
	}
	procChannel <- cmd
	if err := cmd.Wait(); err != nil {
		log.Println(err)
		return
	}
}

func procHandler() {
	for {
		select {
		case proc := <-procChannel:
			procsLock.Lock()
			procs = append(procs, proc)
			procsLock.Unlock()
		}
	}
}

// Kill kills all processes launched by bitcoin.
func Kill() {
	procsLock.Lock()
	defer procsLock.Unlock()
	for _, proc := range procs {
		proc.Process.Kill()
	}
	procs = nil
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

		time.Sleep(2 * time.Second)

		btcdHomeDir := btcutil.AppDataDir("btcd", false)
		certs, err := ioutil.ReadFile(filepath.Join(btcdHomeDir, "rpc.cert"))
		if err != nil {
			cerr = err
			return
		}
		connCfg := &btcrpcclient.ConnConfig{
			Host:         "localhost:18334",
			Endpoint:     "ws",
			User:         RPCUser,
			Pass:         password,
			Certificates: certs,
		}
		_ = connCfg
		client, err = btcrpcclient.New(connCfg, nil) // handlers)
		if err != nil {
			cerr = err
			return
		}
	})
	launchWG.Wait()
	return client, cerr
}
