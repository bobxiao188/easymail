package dkim

import "errors"

var (
	// ErrSignPrivateKeyRequired when there not private key in config
	ErrSignPrivateKeyRequired = errors.New("PrivateKey is required")

	// ErrSignDomainRequired when there is no domain defined in config
	ErrSignDomainRequired = errors.New("Domain is required")

	// ErrSignSelectorRequired when there is no Selector defined in config
	ErrSignSelectorRequired = errors.New("Selector is required")

	// ErrSignHeaderShouldContainsFrom If Headers is specified it should at least contain 'from'
	ErrSignHeaderShouldContainsFrom = errors.New("header must contains 'from' field")

	// ErrSignBadCanonicalization If bad Canonicalization parameter
	ErrSignBadCanonicalization = errors.New("bad Canonicalization parameter")

	// ErrCandNotParsePrivateKey when unable to parse private key
	ErrCandNotParsePrivateKey = errors.New("can not parse private key, check format (pem) and validity")

	// ErrSignBadAlgo Bad algorithm
	ErrSignBadAlgo = errors.New("bad algorithm. Only rsa-sha1 or rsa-sha256 are permitted")

	// ErrBadMailFormat unable to parse mail
	ErrBadMailFormat = errors.New("bad mail format")

	// ErrBadMailFormatHeaders bad headers format
	ErrBadMailFormatHeaders = errors.New("bad mail format found in headers")

	// ErrBadDKimTagLBodyTooShort bad l tag
	ErrBadDKimTagLBodyTooShort = errors.New("bad tag l or bodyLength option. Body length < l value")

	// ErrVerifyInappropriateHashAlgo
	ErrVerifyInappropriateHashAlgo = errors.New("inappropriate hash algorithm")
)
