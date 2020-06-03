package cloudwatch

import (
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gliderlabs/logspout/router"
)

// EC2Info is a subset of the data from the EC2 Metadata Service
type EC2Info struct {
	InstanceID string
	Region     string
}

// NewEC2Info returns a new EC2Info struct with the current InstanceID and
// Region, or nil and an Error if not available.
func NewEC2Info(route *router.Route) (EC2Info, error) {
	_, skipEc2 := route.Options[`NOEC2`]
	if skipEc2 || (os.Getenv(`NOEC2`) != "") {
		return EC2Info{}, nil
	}
	// get my instance ID
	mySession := session.New()
	metadataSvc := ec2metadata.New(mySession)
	if !metadataSvc.Available() {
		log.Println("cloudwatch: WARNING EC2 Metadata service not available")
		return EC2Info{}, nil
	}
	instanceID, err := metadataSvc.GetMetadata(`instance-id`)
	if err != nil {
		return EC2Info{}, fmt.Errorf("ERROR getting instance ID: %s", err)
	}
	region, err := metadataSvc.Region()
	if err != nil {
		return EC2Info{}, fmt.Errorf("ERROR getting EC2 region: %s", err)
	}
	return EC2Info{
		InstanceID: instanceID,
		Region:     region,
	}, nil
}
