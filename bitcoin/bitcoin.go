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
		log.Fatal(err)
	}
	if err := cmd.Wait(); err != nil {
		log.Fatal(err)
	}
}

var launchOnce sync.Once
var launchWG sync.WaitGroup
var client *btcrpcclient.Client

func init() {
	launchWG.Add(1)
}

func NewClient() *btcrpcclient.Client {
	launchOnce.Do(func() {
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
			Host:         "localhost:8334",
			Endpoint:     "ws",
			User:         RPCUser,
			Pass:         password,
			Certificates: certs,
		}
		_ = connCfg
		//time.Sleep(1 * time.Second)
		/*client, err = btcrpcclient.New(connCfg, nil) // handlers)
		if err != nil {
			log.Fatal(err)
		}
		*/
		defer launchWG.Done()
	})
	launchWG.Wait()
	return client
}
