package cloudwatch

import (
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gliderlabs/logspout/router"
)

type EC2Info struct {
	InstanceID string
	Region     string
}

func NewEC2Info(route *router.Route) (EC2Info, error) {
	_, skip_ec2 := route.Options[`NOEC2`]
	if skip_ec2 || (os.Getenv(`NOEC2`) != "") {
		return EC2Info{}, nil
	}
	// get my instance ID
	mySession := session.New()
	metadataSvc := ec2metadata.New(mySession)
	if !metadataSvc.Available() {
		log.Println("cloudwatch: WARNING EC2 Metadata service not available")
		return EC2Info{}, nil
	}
	instance_id, err := metadataSvc.GetMetadata(`instance-id`)
	if err != nil {
		return EC2Info{}, fmt.Errorf("ERROR getting instance ID: %s", err)
	}
	region, err := metadataSvc.Region()
	if err != nil {
		return EC2Info{}, fmt.Errorf("ERROR getting EC2 region: %s", err)
	}
	return EC2Info{
		InstanceID: instance_id,
		Region:     region,
	}, nil
}
