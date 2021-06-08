package massh

import (
	"fmt"
	"golang.org/x/crypto/ssh"
	"os"
	"sync"
	"testing"
	"time"
)

// Credentials are fine to leave here for ease-of-use, as it's an isolated Linux box.
//
// I'm leaving this test (which is being use in examples), here so I can re-use it in the future.

type sshTestParameters struct {
	Hosts []string
	User string
	Password string
}

func TestSshCommandStream(t *testing.T) {
	testParams := sshTestParameters{
		Hosts: []string{"192.168.1.119", "192.168.1.120", "192.168.1.129", "192.168.1.212"},
		User: "u01",
		Password: "password",
	}

	j := &Job{
		Command: "echo \"Hello, World\"",
	}

	sshc := &ssh.ClientConfig{
		User:            testParams.User,
		Auth:            []ssh.AuthMethod{ssh.Password(testParams.Password)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         time.Duration(2) * time.Second,
	}

	cfg := &Config{
		Hosts:      testParams.Hosts,
		SSHConfig:  sshc,
		Job:        j,
		WorkerPool: 10,
	}

	resChan := make(chan Result)

	// This should be the last responsibility from the massh package. Handling the Result channel is up to the user.
	err := cfg.Stream(nil)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	var wg sync.WaitGroup
	// This can probably be cleaner. We're hindered somewhat, I think, by reading a channel from a channel.
	for {
		select {
		case result := <-resChan:
			wg.Add(1)
			go func() {
				if result.Error != nil {
					fmt.Printf("%s: %s\n", result.Host, result.Error)
					wg.Done()
				} else {
					err := readStream(result, &wg)
					if err != nil {
						fmt.Println(err)
					}
				}
			}()
		default:
			if Returned == len(cfg.Hosts) {
				// We want to wait for all goroutines to complete before we declare that the work is finished, as
				// it's possible for us to execute this code before the gofunc above has completed if left unchecked.
				wg.Wait()

				// This should always be the last thing written. Waiting above ensures this.
				fmt.Println("Everything returned.")
				return
			}
		}
	}
}

func readStream(res Result, wg *sync.WaitGroup) error {
	for {
		select {
		case d := <-res.StdOutStream:
			fmt.Printf("%s: %s", res.Host, d)
		case <-res.DoneChannel:
			fmt.Printf("%s: Finished\n", res.Host)
			wg.Done()
		}
	}
}