package acme

import (
	"context"
	"fmt"
	"os"
	"path"
	"time"

	"github.com/dueckminor/home-assistant-addons/go/utils/crypto"
	"golang.org/x/crypto/acme"
)

type Client interface {
	DataDir() string
	IssueCertificate(csr crypto.CSR) (chain crypto.CertificateChain, err error)
}

type ChallengeHandler interface {
	SetChallenge(domain string, challenge string) error
}

type client struct {
	dataDir          string
	client           *acme.Client
	account          *acme.Account
	challengeHandler ChallengeHandler
}

func (c *client) DataDir() string {
	return c.dataDir
}

func (c *client) IssueCertificate(csr crypto.CSR) (chain crypto.CertificateChain, err error) {
	fmt.Println("ACME| Try to create order...")
	ctx := context.Background()

	err = c.GetAccount(ctx)
	if err != nil {
		fmt.Println("ACME| ", err)
		return nil, err
	}
	order, err := c.client.AuthorizeOrder(ctx, acme.DomainIDs(csr.OBJ().DNSNames...))
	if err != nil {
		fmt.Println("ACME| ", err)
		return nil, err
	}
	orderURI := order.URI
	fmt.Println("ACME| Order-URI:", orderURI)

	for {
		fmt.Println("ACME| Order-Status:", order.Status)

		var der [][]byte

		switch order.Status {
		case acme.StatusPending:
			err = c.acceptChallenges(ctx, order)
		case acme.StatusInvalid:
			time.Sleep(time.Second * 5)
		case acme.StatusProcessing:
			time.Sleep(time.Second * 5)
		case acme.StatusReady:
			der, _, err = c.client.CreateOrderCert(ctx, order.FinalizeURL, csr.ASN1(), true)
		case acme.StatusValid:
			der, err = c.client.FetchCert(ctx, order.CertURL, true)
		default:
			return nil, fmt.Errorf("invalid acme order status: %s", order.Status)
		}

		if err != nil {
			fmt.Println("ACME| ", err)
			return nil, err
		}

		if len(der) > 0 {
			return c.derToChain(der)
		}

		fmt.Println("ACME| Refresh-Order:", orderURI)

		order, err = c.client.GetOrder(ctx, orderURI)
		if err != nil {
			fmt.Println("ACME| ", err)
			return nil, err
		}
	}
}

func (c *client) derToChain(der [][]byte) (chain crypto.CertificateChain, err error) {
	for _, asn1 := range der {
		cert, err := crypto.NewCertificateFromASN1(asn1)
		if err != nil {
			return nil, err
		}
		chain = append(chain, cert)
	}
	return chain, nil
}

func (c *client) GetAccount(ctx context.Context) (err error) {
	if c.account != nil {
		return nil
	}
	c.account, err = c.client.GetReg(ctx, "")
	if err != nil {
		fmt.Println("ACME| ", err)
	}
	if c.account != nil {
		return nil
	}
	acct := &acme.Account{}

	c.account, err = c.client.Register(ctx, acct, acme.AcceptTOS)
	if err != nil {
		return err
	}
	return nil
}

func (c *client) acceptChallenges(ctx context.Context, order *acme.Order) (err error) {
	for _, authURL := range order.AuthzURLs {
		auth, err := c.client.GetAuthorization(ctx, authURL)
		if err != nil {
			fmt.Println("ACME| ", err)
			return err
		}
		fmt.Println("ACME| Identifier:", auth.Identifier.Value)
		for _, challenge := range auth.Challenges {
			fmt.Println("ACME| - Type:", challenge.Type)
			fmt.Println("ACME|   URI:", challenge.URI)
			fmt.Println("ACME|   Token:", challenge.Token)
			fmt.Println("ACME|   Status:", challenge.Status)

			if challenge.Status != acme.StatusPending {
				continue
			}
			record, err := c.client.DNS01ChallengeRecord(challenge.Token)
			if err != nil {
				fmt.Println("ACME| ", err)
				return err
			}
			fmt.Println("ACME|   Record:", record)
			c.challengeHandler.SetChallenge(auth.Identifier.Value, record)
			challenge, err = c.client.Accept(ctx, challenge)
			if err != nil {
				fmt.Println("ACME| ", err)
				return err
			}
			fmt.Println("ACME|   New-Status:", challenge.Status)
		}
	}

	return nil
}

func NewClient(dataDir string, challengeHandler ChallengeHandler) (c Client, err error) {
	err = os.MkdirAll(dataDir, 0700)
	if err != nil {
		return nil, err
	}
	key, err := crypto.GetOrCreatePrivateKeyFile(path.Join(dataDir, "account_key.pem"))
	if err != nil {
		return nil, err
	}
	return &client{
		dataDir:          dataDir,
		client:           &acme.Client{Key: key},
		challengeHandler: challengeHandler,
	}, nil
}
