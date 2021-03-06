package mfa

import (
	"encoding/base32"
	"fmt"
	"strings"
	"time"

	"github.com/99designs/aws-vault/mfa/device"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/pkg/errors"
)

// MFA combines IAM config with a MFA device
type MFA struct {
	iam    *iam.IAM
	device device.ReaderManager
	sts    *sts.STS
}

// New initializes a AWS virtual MFA device as target
func New(sess *session.Session, d device.ReaderManager) (*MFA, error) {
	return &MFA{
		iam:    iam.New(sess),
		sts:    sts.New(sess),
		device: d,
	}, nil
}

// Add adds a Yubikey as a virtual MFA
func (m *MFA) Add(username string) (*string, []byte, error) {
	serial, secret, err := m.create(username)

	if err != nil {
		return nil, nil, err
	}

	if err := m.enable(username, serial, secret); err != nil {
		return nil, nil, err
	}

	return serial, secret, nil
}

// Delete removes a virtual MFA from the source including it's association with
// the given IAM username
func (m *MFA) Delete(username string) error {
	res, err := m.sts.GetCallerIdentity(&sts.GetCallerIdentityInput{})

	if err != nil {
		return errors.Wrap(err, "failed to determine serial number for device deletion")
	}

	serial, err := callerIdentityToSerial(res.Arn)

	if err != nil {
		return err
	}

	err = m.deactivate(username, &serial)
	if err != nil {
		return err
	}

	if err := m.delete(&serial); err != nil {
		return err
	}

	return nil
}

// create creates the virtual MFA device
func (m *MFA) create(username string) (*string, []byte, error) {
	res, err := m.iam.CreateVirtualMFADevice(&iam.CreateVirtualMFADeviceInput{
		VirtualMFADeviceName: &username,
	})

	if err != nil {
		return nil, nil, errors.Wrap(err, "error creating virtual device for iam user")
	}

	secret, err := base32.StdEncoding.DecodeString(string(res.VirtualMFADevice.Base32StringSeed))

	if err != nil {
		return nil, nil, errors.Wrap(err, "error decoding secret")
	}

	return res.VirtualMFADevice.SerialNumber, secret, nil
}

func (m *MFA) enable(username string, serial *string, secret []byte) error {
	name, err := SerialToName(serial)
	if err != nil {
		return err
	}

	if err = m.device.Add(name, secret); err != nil {
		return errors.Wrapf(err, "error adding source %s %s", name, m.device.Name())
	}

	otp1, err := m.device.GetOTP(time.Now(), name)

	if err != nil {
		return errors.Wrap(err, "error getting first otp")
	}

	otp2, err := m.device.GetOTP(time.Now().Add(30*time.Second), name)

	if err != nil {
		return errors.Wrap(err, "error getting second otp")
	}

	fmt.Printf("Yubikey virtual mfa device enabled with codes %s and %s\n", otp1, otp2)

	if _, err := m.iam.EnableMFADevice(&iam.EnableMFADeviceInput{
		AuthenticationCode1: &otp1,
		AuthenticationCode2: &otp2,
		SerialNumber:        serial,
		UserName:            &username,
	}); err != nil {
		return errors.Wrap(err, "error enabling Yubikey as virtual mfa device")
	}

	return nil
}

// deactivate deactivates the virtual MFA device and removes it from the source
func (m *MFA) deactivate(username string, serial *string) error {
	_, err := m.iam.DeactivateMFADevice(&iam.DeactivateMFADeviceInput{
		SerialNumber: serial,
		UserName:     &username,
	})

	if err != nil {
		awsErr, ok := err.(awserr.Error)

		if !ok || ok && awsErr.Code() != "NoSuchEntity" {
			return errors.Wrapf(err, "failed to deactivate virtual AWS MFA device with serial %q", *serial)
		}
	}

	return nil
}

// delete deletes the virtual MFA device
func (m *MFA) delete(serial *string) error {
	_, err := m.iam.DeleteVirtualMFADevice(&iam.DeleteVirtualMFADeviceInput{
		SerialNumber: serial,
	})

	if err != nil {
		awsErr, ok := err.(awserr.Error)

		if !ok || ok && awsErr.Code() != "NoSuchEntity" {
			return errors.Wrapf(err, "failed to delete virtual AWS MFA device with serial %q", *serial)
		}
	}

	name, err := SerialToName(serial)

	if err != nil {
		return err
	}

	if err := m.device.Delete(name); err != nil {
		fmt.Println(err) // underpowered or cred doesn't exist?
		return errors.Wrapf(err, "failed to remove %q from source %q", name, m.device.Name())
	}

	return nil
}

// callerIdentityToSerial converts a caller identity ARN to a MFA serial
func callerIdentityToSerial(i *string) (string, error) {
	a, err := arn.Parse(*i)

	if err != nil {
		return "", errors.Wrapf(err, "failed to parse %q as ARN", *i)
	}

	return strings.Replace(a.String(), ":user/", ":mfa/", 1), nil
}

// SerialToName converts a MFA serial to a issuer:account name string that displays nicely in the
// Yubico Authenticator app as
// --------------------------------------------
// |  issuer (substring before first :)       |
// |  otp here (after Yubikey touch possibly) |
// |  account name (substring after first :)  |
// --------------------------------------------
func SerialToName(i *string) (string, error) {
	a, err := arn.Parse(*i)

	if err != nil {
		return "", errors.Wrapf(err, "unable to parse arn: %s", *i)
	}

	return strings.Join([]string{
		"aws",      // issuer
		a.String(), // account name
	}, ":"), nil
}
