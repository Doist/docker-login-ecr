// Command docker-login-ecr logs docker client into AWS ECR
package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"unicode/utf8"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecr"
)

func main() {
	flag.Parse() // to handle -h/--help
	if err := run(); err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}
}

func run() error {
	sess, err := session.NewSession()
	if err != nil {
		return err
	}
	var svc *ecr.ECR
	switch meta, err := ec2metadata.New(sess).GetInstanceIdentityDocument(); err {
	case nil:
		svc = ecr.New(sess, aws.NewConfig().WithRegion(meta.Region))
	default:
		svc = ecr.New(sess)
	}
	out, err := svc.GetAuthorizationToken(nil)
	if err != nil {
		return err
	}
	if l := len(out.AuthorizationData); l != 1 {
		return fmt.Errorf("ECR returned %d tokens, want exactly 1", l)
	}
	endpoint := *out.AuthorizationData[0].ProxyEndpoint
	if !strings.HasPrefix(endpoint, "https:") {
		return fmt.Errorf("endpoint is of the wrong format: %q", endpoint)
	}
	b, err := base64.StdEncoding.DecodeString(*out.AuthorizationData[0].AuthorizationToken)
	if err != nil {
		return err
	}
	if !utf8.Valid(b) {
		return fmt.Errorf("decoded login:password pair is not valid utf8")
	}
	fs := strings.SplitN(string(b), ":", 2)
	if len(fs) != 2 {
		return fmt.Errorf("decoded login:password pair is of the wrong format: %q", b)
	}
	login, password := fs[0], fs[1]
	_ = password
	args := []string{"login", "--username", login, "--password-stdin", endpoint}
	fmt.Println("running: docker", strings.Join(args, " "))
	cmd := exec.Command("docker", args...)
	cmd.Stdin = strings.NewReader(password)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func init() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Command %s logs docker client into AWS ECR\n", os.Args[0])
	}
}
