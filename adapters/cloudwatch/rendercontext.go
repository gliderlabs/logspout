package cloudwatch

import (
	"fmt"
)

// RenderContext defines the info that can be used in
// LogGroup and LogStream names.
type RenderContext struct {
	Host       string            // container host name
	Env        map[string]string // container ENV
	Labels     map[string]string // container Labels
	Name       string            // container Name
	ID         string            // container ID
	LoggerHost string            // hostname of logging container (os.Hostname)
	InstanceID string            // EC2 Instance ID
	Region     string            // EC2 region
}

// Lbl renders a label value based on a given key
func (r *RenderContext) Lbl(key string) (string, error) {
	if val, exists := r.Labels[key]; exists {
		return val, nil
	}
	return "", fmt.Errorf("ERROR reading container label %s", key)
}
