package snmp

import (
	"testing"

	"github.com/gosnmp/gosnmp"
)

func TestNewClient(t *testing.T) {
	t.Run("should create a v2c client by default", func(t *testing.T) {
		client, err := NewClient(Config{Host: "localhost", Port: 161, Community: "public"})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if client.snmp.Version != gosnmp.Version2c {
			t.Errorf("expected version %v, got %v", gosnmp.Version2c, client.snmp.Version)
		}
		if client.snmp.Community != "public" {
			t.Errorf("expected community 'public', got %s", client.snmp.Community)
		}
	})

	t.Run("should create a v3 client with correct security parameters", func(t *testing.T) {
		config := Config{
			Host:             "localhost",
			Port:             161,
			Version:          "v3",
			ContextName:      "mycontext",
			SecurityLevel:    "authPriv",
			SecurityUsername: "myuser",
			AuthProtocol:     "SHA",
			AuthPassword:     "authpass",
			PrivProtocol:     "AES",
			PrivPassword:     "privpass",
		}

		client, err := NewClient(config)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if client.snmp.Version != gosnmp.Version3 {
			t.Errorf("expected version %v, got %v", gosnmp.Version3, client.snmp.Version)
		}

		if client.snmp.ContextName != "mycontext" {
			t.Errorf("expected context name 'mycontext', got %s", client.snmp.ContextName)
		}

		if client.snmp.MsgFlags != gosnmp.AuthPriv {
			t.Errorf("expected msg flags %v, got %v", gosnmp.AuthPriv, client.snmp.MsgFlags)
		}

		usmParams, ok := client.snmp.SecurityParameters.(*gosnmp.UsmSecurityParameters)
		if !ok {
			t.Fatal("expected security parameters to be of type UsmSecurityParameters")
		}

		if usmParams.UserName != "myuser" {
			t.Errorf("expected username 'myuser', got %s", usmParams.UserName)
		}

		if usmParams.AuthenticationProtocol != gosnmp.SHA {
			t.Errorf("expected auth protocol %v, got %v", gosnmp.SHA, usmParams.AuthenticationProtocol)
		}

		if usmParams.AuthenticationPassphrase != "authpass" {
			t.Errorf("expected auth passphrase 'authpass', got %s", usmParams.AuthenticationPassphrase)
		}

		if usmParams.PrivacyProtocol != gosnmp.AES {
			t.Errorf("expected privacy protocol %v, got %v", gosnmp.AES, usmParams.PrivacyProtocol)
		}

		if usmParams.PrivacyPassphrase != "privpass" {
			t.Errorf("expected privacy passphrase 'privpass', got %s", usmParams.PrivacyPassphrase)
		}
	})

	t.Run("should accept extended AES privacy protocols", func(t *testing.T) {
		config := Config{
			Host:             "localhost",
			Port:             161,
			Version:          "v3",
			SecurityLevel:    "authPriv",
			SecurityUsername: "user",
			AuthProtocol:     "SHA256",
			AuthPassword:     "authpass",
			PrivProtocol:     "AES256C",
			PrivPassword:     "privpass",
		}

		client, err := NewClient(config)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		usmParams, ok := client.snmp.SecurityParameters.(*gosnmp.UsmSecurityParameters)
		if !ok {
			t.Fatal("expected security parameters to be of type UsmSecurityParameters")
		}

		if usmParams.PrivacyProtocol != gosnmp.AES256C {
			t.Errorf("expected privacy protocol %v, got %v", gosnmp.AES256C, usmParams.PrivacyProtocol)
		}
	})
}
