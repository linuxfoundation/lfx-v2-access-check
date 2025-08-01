// Code generated by goa v3.21.5, DO NOT EDIT.
//
// access-svc HTTP client CLI support package
//
// Command:
// $ goa gen github.com/linuxfoundation/lfx-v2-access-check/design

package cli

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	accesssvcc "github.com/linuxfoundation/lfx-v2-access-check/gen/http/access_svc/client"
	goahttp "goa.design/goa/v3/http"
	goa "goa.design/goa/v3/pkg"
)

// UsageCommands returns the set of commands and sub-commands using the format
//
//	command (subcommand1|subcommand2|...)
func UsageCommands() string {
	return `access-svc (check-access|readyz|livez)
`
}

// UsageExamples produces an example of a valid invocation of the CLI tool.
func UsageExamples() string {
	return os.Args[0] + ` access-svc check-access --body '{
      "requests": [
         "project:123:read",
         "committee:456:write"
      ]
   }' --version "1" --bearer-token "Nostrum adipisci magni quisquam voluptatem."` + "\n" +
		""
}

// ParseEndpoint returns the endpoint and payload as specified on the command
// line.
func ParseEndpoint(
	scheme, host string,
	doer goahttp.Doer,
	enc func(*http.Request) goahttp.Encoder,
	dec func(*http.Response) goahttp.Decoder,
	restore bool,
) (goa.Endpoint, any, error) {
	var (
		accessSvcFlags = flag.NewFlagSet("access-svc", flag.ContinueOnError)

		accessSvcCheckAccessFlags           = flag.NewFlagSet("check-access", flag.ExitOnError)
		accessSvcCheckAccessBodyFlag        = accessSvcCheckAccessFlags.String("body", "REQUIRED", "")
		accessSvcCheckAccessVersionFlag     = accessSvcCheckAccessFlags.String("version", "REQUIRED", "")
		accessSvcCheckAccessBearerTokenFlag = accessSvcCheckAccessFlags.String("bearer-token", "REQUIRED", "")

		accessSvcReadyzFlags = flag.NewFlagSet("readyz", flag.ExitOnError)

		accessSvcLivezFlags = flag.NewFlagSet("livez", flag.ExitOnError)
	)
	accessSvcFlags.Usage = accessSvcUsage
	accessSvcCheckAccessFlags.Usage = accessSvcCheckAccessUsage
	accessSvcReadyzFlags.Usage = accessSvcReadyzUsage
	accessSvcLivezFlags.Usage = accessSvcLivezUsage

	if err := flag.CommandLine.Parse(os.Args[1:]); err != nil {
		return nil, nil, err
	}

	if flag.NArg() < 2 { // two non flag args are required: SERVICE and ENDPOINT (aka COMMAND)
		return nil, nil, fmt.Errorf("not enough arguments")
	}

	var (
		svcn string
		svcf *flag.FlagSet
	)
	{
		svcn = flag.Arg(0)
		switch svcn {
		case "access-svc":
			svcf = accessSvcFlags
		default:
			return nil, nil, fmt.Errorf("unknown service %q", svcn)
		}
	}
	if err := svcf.Parse(flag.Args()[1:]); err != nil {
		return nil, nil, err
	}

	var (
		epn string
		epf *flag.FlagSet
	)
	{
		epn = svcf.Arg(0)
		switch svcn {
		case "access-svc":
			switch epn {
			case "check-access":
				epf = accessSvcCheckAccessFlags

			case "readyz":
				epf = accessSvcReadyzFlags

			case "livez":
				epf = accessSvcLivezFlags

			}

		}
	}
	if epf == nil {
		return nil, nil, fmt.Errorf("unknown %q endpoint %q", svcn, epn)
	}

	// Parse endpoint flags if any
	if svcf.NArg() > 1 {
		if err := epf.Parse(svcf.Args()[1:]); err != nil {
			return nil, nil, err
		}
	}

	var (
		data     any
		endpoint goa.Endpoint
		err      error
	)
	{
		switch svcn {
		case "access-svc":
			c := accesssvcc.NewClient(scheme, host, doer, enc, dec, restore)
			switch epn {
			case "check-access":
				endpoint = c.CheckAccess()
				data, err = accesssvcc.BuildCheckAccessPayload(*accessSvcCheckAccessBodyFlag, *accessSvcCheckAccessVersionFlag, *accessSvcCheckAccessBearerTokenFlag)
			case "readyz":
				endpoint = c.Readyz()
			case "livez":
				endpoint = c.Livez()
			}
		}
	}
	if err != nil {
		return nil, nil, err
	}

	return endpoint, data, nil
}

// accessSvcUsage displays the usage of the access-svc command and its
// subcommands.
func accessSvcUsage() {
	fmt.Fprintf(os.Stderr, `LFX Access Check Service
Usage:
    %[1]s [globalflags] access-svc COMMAND [flags]

COMMAND:
    check-access: Check access permissions for resource-action pairs
    readyz: Check if service is ready
    livez: Check if service is alive

Additional help:
    %[1]s access-svc COMMAND --help
`, os.Args[0])
}
func accessSvcCheckAccessUsage() {
	fmt.Fprintf(os.Stderr, `%[1]s [flags] access-svc check-access -body JSON -version STRING -bearer-token STRING

Check access permissions for resource-action pairs
    -body JSON: 
    -version STRING: 
    -bearer-token STRING: 

Example:
    %[1]s access-svc check-access --body '{
      "requests": [
         "project:123:read",
         "committee:456:write"
      ]
   }' --version "1" --bearer-token "Nostrum adipisci magni quisquam voluptatem."
`, os.Args[0])
}

func accessSvcReadyzUsage() {
	fmt.Fprintf(os.Stderr, `%[1]s [flags] access-svc readyz

Check if service is ready

Example:
    %[1]s access-svc readyz
`, os.Args[0])
}

func accessSvcLivezUsage() {
	fmt.Fprintf(os.Stderr, `%[1]s [flags] access-svc livez

Check if service is alive

Example:
    %[1]s access-svc livez
`, os.Args[0])
}
