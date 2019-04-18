// Copyright (C) 2019 Storj Labs, Inc.
// See LICENSE for copying information.

package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zeebo/errs"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh/terminal"

	"storj.io/storj/internal/fpath"
	"storj.io/storj/pkg/cfgstruct"
	"storj.io/storj/pkg/process"
)

var (
	setupCmd = &cobra.Command{
		Use:         "setup",
		Short:       "Create an uplink config file",
		RunE:        cmdSetup,
		Annotations: map[string]string{"type": "setup"},
	}
	setupCfg    UplinkFlags
	confDir     string
	identityDir string
	isDev       bool

	// Error is the default uplink setup errs class
	Error = errs.Class("uplink setup error")
)

func init() {
	defaultConfDir := fpath.ApplicationDir("storj", "uplink")
	defaultIdentityDir := fpath.ApplicationDir("storj", "identity", "uplink")
	cfgstruct.SetupFlag(zap.L(), RootCmd, &confDir, "config-dir", defaultConfDir, "main directory for uplink configuration")
	cfgstruct.SetupFlag(zap.L(), RootCmd, &identityDir, "identity-dir", defaultIdentityDir, "main directory for uplink identity credentials")
	cfgstruct.DevFlag(RootCmd, &isDev, false, "use development and test configuration settings")
	RootCmd.AddCommand(setupCmd)
	cfgstruct.BindSetup(setupCmd.Flags(), &setupCfg, isDev, cfgstruct.ConfDir(confDir), cfgstruct.IdentityDir(identityDir))
}

func cmdSetup(cmd *cobra.Command, args []string) (err error) {
	// Ensure use the default port if the user only specifies a host.
	err = ApplyDefaultHostAndPortToAddrFlag(cmd, "satellite-addr")
	if err != nil {
		return err
	}

	setupDir, err := filepath.Abs(confDir)
	if err != nil {
		return err
	}

	valid, _ := fpath.IsValidSetupDir(setupDir)
	if !valid {
		return fmt.Errorf("uplink configuration already exists (%v)", setupDir)
	}

	err = os.MkdirAll(setupDir, 0700)
	if err != nil {
		return err
	}

	var (
		encKeyFilepath = filepath.Join(setupDir, ".enc.key")
		encKey         []byte
		override       = map[string]interface{}{
			"enc.key-filepath": encKeyFilepath,
		}
	)
	if !setupCfg.NonInteractive {
		_, err = fmt.Print("Enter your Satellite address: ")
		if err != nil {
			return err
		}
		var satelliteAddress string
		_, err = fmt.Scanln(&satelliteAddress)
		if err != nil {
			return err
		}

		// TODO add better validation
		if satelliteAddress == "" {
			return errs.New("Satellite address cannot be empty")
		}

		satelliteAddress, err = ApplyDefaultHostAndPortToAddr(satelliteAddress, cmd.Flags().Lookup("satellite-addr").Value.String())
		if err != nil {
			return err
		}
		override["satellite-addr"] = satelliteAddress

		_, err = fmt.Print("Enter your API key: ")
		if err != nil {
			return err
		}
		var apiKey string
		_, err = fmt.Scanln(&apiKey)
		if err != nil {
			return err
		}

		if apiKey == "" {
			return errs.New("API key cannot be empty")
		}
		override["api-key"] = apiKey

		_, err = fmt.Print("Enter your encryption passphrase: ")
		if err != nil {
			return err
		}
		encKey, err = terminal.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			return err
		}
		_, err = fmt.Println()
		if err != nil {
			return err
		}

		_, err = fmt.Print("Enter your encryption passphrase again: ")
		if err != nil {
			return err
		}
		repeatedEncKey, err := terminal.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			return err
		}
		_, err = fmt.Println()
		if err != nil {
			return err
		}

		if !bytes.Equal(encKey, repeatedEncKey) {
			return errs.New("encryption passphrases doesn't match")
		}

		if len(encKey) == 0 {
			_, err = fmt.Println("Warning: Encryption passphrase is empty!")
			if err != nil {
				return err
			}
		} else {
			err = SaveEncryptionKey(encKey, encKeyFilepath)
			if err != nil {
				return err
			}
		}
	}

	return process.SaveConfigWithAllDefaults(
		cmd.Flags(), filepath.Join(setupDir, process.DefaultConfFilename), override)
}

// ApplyDefaultHostAndPortToAddrFlag applies the default host and/or port if either is missing in the specified flag name.
func ApplyDefaultHostAndPortToAddrFlag(cmd *cobra.Command, flagName string) error {
	flag := cmd.Flags().Lookup(flagName)
	if flag == nil {
		// No flag found for us to handle.
		return nil
	}

	address, err := ApplyDefaultHostAndPortToAddr(flag.Value.String(), flag.DefValue)
	if err != nil {
		return Error.Wrap(err)
	}

	if flag.Value.String() == address {
		// Don't trip the flag set bit
		return nil
	}

	return Error.Wrap(flag.Value.Set(address))
}

// ApplyDefaultHostAndPortToAddr applies the default host and/or port if either is missing in the specified address.
func ApplyDefaultHostAndPortToAddr(address, defaultAddress string) (string, error) {
	defaultHost, defaultPort, err := net.SplitHostPort(defaultAddress)
	if err != nil {
		return "", Error.Wrap(err)
	}

	addressParts := strings.Split(address, ":")
	numberOfParts := len(addressParts)

	if numberOfParts > 1 && len(addressParts[0]) > 0 && len(addressParts[1]) > 0 {
		// address is host:port so skip applying any defaults.
		return address, nil
	}

	// We are missing a host:port part. Figure out which part we are missing.
	indexOfPortSeparator := strings.Index(address, ":")
	lengthOfFirstPart := len(addressParts[0])

	if indexOfPortSeparator < 0 {
		if lengthOfFirstPart == 0 {
			// address is blank.
			return defaultAddress, nil
		}
		// address is host
		return net.JoinHostPort(addressParts[0], defaultPort), nil
	}

	if indexOfPortSeparator == 0 {
		// address is :1234
		return net.JoinHostPort(defaultHost, addressParts[1]), nil
	}

	// address is host:
	return net.JoinHostPort(addressParts[0], defaultPort), nil
}

// SaveEncryptionKey saves the key in a new file which will be stored in
// filepath.
// It returns an error if the directory doesn't exist, the file already exists
// or there is an I/O error.
func SaveEncryptionKey(key []byte, filepath string) (err error) {
	f, err := os.OpenFile(filepath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		if os.IsNotExist(err) {
			return errors.New("directory path doesn't exist")
		}

		if os.IsExist(err) {
			return errors.New("file key already exists")
		}

		return err
	}

	defer func() {
		if err == nil {
			err = f.Close()
		} else {
			_ = f.Close()
		}
	}()

	_, err = f.Write(key)
	if err != nil {
		return err
	}

	return f.Chmod(0400)
}
