package util

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net/http"
	"os"
	"time"

	MQTT "github.com/eclipse/paho.mqtt.golang"
	"k8s.io/klog/v2"

	eventconfig "github.com/kubeedge/kubeedge/edge/pkg/eventbus/config"
)

var (
	// TokenWaitTime to wait
	TokenWaitTime = 120 * time.Second
	// Loop connect to wait
	LoopConnectPeriord = 5 * time.Second
)

// CheckKeyExist check dis info format
func CheckKeyExist(keys []string, disinfo map[string]interface{}) error {
	for _, v := range keys {
		_, ok := disinfo[v]
		if !ok {
			klog.Errorf("key: %s not found", v)
			return errors.New("key not found")
		}
	}
	return nil
}

// CheckClientToken checks token is right
func CheckClientToken(token MQTT.Token) (bool, error) {
	if token.Wait() && token.Error() != nil {
		return false, token.Error()
	}
	return true, nil
}

// PathExist check file exists or not
func PathExist(path string) bool {
	_, err := os.Stat(path)
	return err == nil || os.IsExist(err)
}

// HubClientInit create mqtt client config
func HubClientInit(server, clientID, username, password string) *MQTT.ClientOptions {
	opts := MQTT.NewClientOptions().AddBroker(server).SetClientID(clientID).SetCleanSession(true)
	if username != "" {
		opts.SetUsername(username)
		if password != "" {
			opts.SetPassword(password)
		}
	}

	klog.V(4).Infof("Start to set TLS configuration for MQTT client")
	tlsConfig := &tls.Config{}
	if eventconfig.Config.TLS.Enable {
		cert, err := tls.LoadX509KeyPair(eventconfig.Config.TLS.TLSMqttCertFile, eventconfig.Config.TLS.TLSMqttPrivateKeyFile)
		if err != nil {
			klog.Errorf("Failed to load x509 key pair: %v", err)
			return nil
		}

		caCert, err := os.ReadFile(eventconfig.Config.TLS.TLSMqttCAFile)
		if err != nil {
			klog.Errorf("Failed to read TLSMqttCAFile")
			return nil
		}

		pool := x509.NewCertPool()
		if ok := pool.AppendCertsFromPEM(caCert); !ok {
			klog.Errorf("Cannot parse the certificates")
			return nil
		}

		tlsConfig = &tls.Config{
			RootCAs:            pool,
			Certificates:       []tls.Certificate{cert},
			InsecureSkipVerify: false,
		}
	} else {
		tlsConfig = &tls.Config{InsecureSkipVerify: true, ClientAuth: tls.NoClientCert}
	}
	opts.SetTLSConfig(tlsConfig)
	klog.V(4).Infof("set TLS configuration for MQTT client successfully")

	return opts
}

// LoopConnect connect to mqtt server
func LoopConnect(clientID string, client MQTT.Client) {
	for {
		klog.Infof("start connect to mqtt server with client id: %s", clientID)
		token := client.Connect()
		klog.Infof("client %s isconnected: %v", clientID, client.IsConnected())
		if rs, err := CheckClientToken(token); !rs {
			klog.Errorf("connect error: %v", err)
		} else {
			return
		}
		time.Sleep(LoopConnectPeriord)
	}
}

const remoteAPIURL = "https://remote-server/api"

func CallRemoteAPI(payload []byte) {
	// Initialise client
	client := &http.Client{}
	// create the request
	req, err :=http.NewRequest("POST", remoteAPIURL, bytes.NewBuffer(payload))
	if err != nil {
		klog.Errorf("Error creating request: %v", err)
		return
	}
	// make the header
	req.Header.Set("Content-Type", "application/json")
	// send and close the request
	res, err := client.Do(req)
	if err != nil {
		klog.Errorf("Error sending request: %v", err)
		return
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		klog.Errorf("Error response: %v", res.Status)
		return
	}

	klog.Infof("Success response: %v", res.Status)
}
