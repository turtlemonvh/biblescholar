package server

import (
	"bytes"
	"crypto"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Jeffail/gabs"
	"github.com/gin-gonic/gin"

	log "github.com/sirupsen/logrus"
)

const (
	BibleScholarAppId string = "amzn1.ask.skill.30c203ed-c0f7-432e-bcdb-23ee1ece38ab"
	appIdPath         string = "session.application.applicationId"
)

// Only for error responses
func logAndSetResponse(c *gin.Context, status int, ctx log.Fields) {
	c.JSON(status, ctx)
	log.WithFields(ctx).Error("Error validating HTTP certificate")
}

func (s *ServerConfig) validateAppId(c *gin.Context, req *gabs.Container) error {
	sentAppId, ok := req.Path(appIdPath).Data().(string)
	if !ok {
		err := fmt.Errorf("Missing required field: '%s'", appIdPath)
		logAndSetResponse(c, http.StatusBadRequest, log.Fields{
			"error": err.Error(),
		})
		return err
	}
	if sentAppId != BibleScholarAppId {
		err := fmt.Errorf("Request sent with the wring app id: sent: %s, expected: %s", sentAppId, BibleScholarAppId)
		logAndSetResponse(c, http.StatusBadRequest, log.Fields{
			"error": err,
		})
		return err
	}
	return nil
}

// Required to be called in production
// From: https://github.com/mikeflynn/go-alexa/blob/master/skillserver/skillserver.go#L154
func (s *ServerConfig) verifyRequestIsAlexa(c *gin.Context) error {
	r := c.Request

	certURL := r.Header.Get("SignatureCertChainUrl")

	// Verify certificate URL
	if !verifyCertURL(certURL) {
		err := fmt.Errorf("Invalid cert URL: '%v'", certURL)
		logAndSetResponse(c, http.StatusUnauthorized, log.Fields{
			"error": err.Error(),
		})
		return err
	}

	// Fetch certificate data
	certContents, err := readCert(certURL)
	if err != nil {
		err := fmt.Errorf("Unable to read certificate for url: %v", certURL)
		logAndSetResponse(c, http.StatusUnauthorized, log.Fields{
			"error": err.Error(),
		})
		return err
	}

	// Decode certificate data
	block, _ := pem.Decode(certContents)
	if block == nil {
		err := fmt.Errorf("Failed to parse certificate PEM for url: %v", certURL)
		logAndSetResponse(c, http.StatusUnauthorized, log.Fields{
			"error": err.Error(),
		})
		return err
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		err := fmt.Errorf("Failed to parse certificate for url: %v", certURL)
		logAndSetResponse(c, http.StatusUnauthorized, log.Fields{
			"error": err.Error(),
		})
		return err
	}

	// Check the certificate date
	if time.Now().Unix() < cert.NotBefore.Unix() || time.Now().Unix() > cert.NotAfter.Unix() {
		err := fmt.Errorf("Amazon certificate expired")
		logAndSetResponse(c, http.StatusUnauthorized, log.Fields{
			"error": err.Error(),
		})
		return err
	}

	// Check the certificate alternate names
	foundName := false
	for _, altName := range cert.Subject.Names {
		if altName.Value == "echo-api.amazon.com" {
			foundName = true
		}
	}

	if !foundName {
		err := fmt.Errorf("Amazon certificate invalid.")
		logAndSetResponse(c, http.StatusUnauthorized, log.Fields{
			"error": err.Error(),
		})
		return err
	}

	// Verify the key
	publicKey := cert.PublicKey
	encryptedSig, _ := base64.StdEncoding.DecodeString(r.Header.Get("Signature"))

	// Make the request body SHA1 and verify the request with the public key
	var bodyBuf bytes.Buffer
	hash := sha1.New()
	_, err = io.Copy(hash, io.TeeReader(r.Body, &bodyBuf))
	if err != nil {
		err := fmt.Errorf("Internal Error.")
		logAndSetResponse(c, http.StatusInternalServerError, log.Fields{
			"error": err.Error(),
		})
		return err
	}

	r.Body = ioutil.NopCloser(&bodyBuf)

	err = rsa.VerifyPKCS1v15(publicKey.(*rsa.PublicKey), crypto.SHA1, hash.Sum(nil), encryptedSig)
	if err != nil {
		err := fmt.Errorf("Signature match failed.")
		logAndSetResponse(c, http.StatusUnauthorized, log.Fields{
			"error": err.Error(),
		})
		return err
	}

	return nil
}

func readCert(certURL string) ([]byte, error) {
	cert, err := http.Get(certURL)
	if err != nil {
		return nil, errors.New("Could not download Amazon cert file.")
	}
	defer cert.Body.Close()
	certContents, err := ioutil.ReadAll(cert.Body)
	if err != nil {
		return nil, errors.New("Could not read Amazon cert file.")
	}

	return certContents, nil
}

func verifyCertURL(path string) bool {
	link, _ := url.Parse(path)

	if link.Scheme != "https" {
		return false
	}

	if link.Host != "s3.amazonaws.com" && link.Host != "s3.amazonaws.com:443" {
		return false
	}

	if !strings.HasPrefix(link.Path, "/echo.api/") {
		return false
	}

	return true
}
