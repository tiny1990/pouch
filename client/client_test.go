package client

import (
	"crypto/tls"
	"fmt"
	"net/url"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	testHost = "unix:///var/run/pouchd.sock"
)

func TestNewAPIClient(t *testing.T) {
	assert := assert.New(t)
	kvs := map[string]bool{
		"":                      false,
		"foobar":                true,
		"tcp://localhost:2476":  false,
		"http://localhost:2476": false,
	}

	for host, expectError := range kvs {
		cli, err := NewAPIClient(host, TLSConfig{})
		if expectError {
			assert.Error(err, fmt.Sprintf("test data: %v", host))
		} else {
			assert.NoError(err, fmt.Sprintf("test data %v: %v", host, err))
		}

		t.Logf("client info %+v", cli)
	}
}

func TestParseHost(t *testing.T) {
	assert := assert.New(t)
	type parsed struct {
		host           string
		expectError    bool
		expectBasePath string
		expectAddr     string
	}

	parseds := []parsed{
		{host: testHost, expectError: false, expectBasePath: "http://d", expectAddr: "/var/run/pouchd.sock"},
		{host: "tcp://localhost:1234", expectError: false, expectBasePath: "http://localhost:1234", expectAddr: "localhost:1234"},
		{host: "http://localhost:5678", expectError: false, expectBasePath: "http://localhost:5678", expectAddr: "localhost:5678"},
		{host: "foo:bar", expectError: true, expectBasePath: "", expectAddr: ""},
		{host: "", expectError: true, expectBasePath: "", expectAddr: ""},
	}

	for _, p := range parseds {
		_, basePath, addr, err := parseHost(p.host)
		if p.expectError {
			assert.Error(err, fmt.Sprintf("test data %v", p.host))
		} else {
			assert.NoError(err, fmt.Sprintf("test data %v", p.host))
		}

		assert.Equal(basePath, p.expectBasePath)
		assert.Equal(addr, p.expectAddr)
	}
}

func Test_generateBaseURL(t *testing.T) {
	type args struct {
		u   *url.URL
		tls TLSConfig
	}
	tests := []struct {
		name string
		args args
		want string
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := generateBaseURL(tt.args.u, tt.args.tls); got != tt.want {
				t.Errorf("generateBaseURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGenTLSConfig(t *testing.T) {
	type args struct {
		key  string
		cert string
		ca   string
	}
	tests := []struct {
		name    string
		args    args
		want    *tls.Config
		wantErr bool
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GenTLSConfig(tt.args.key, tt.args.cert, tt.args.ca)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenTLSConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GenTLSConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}
