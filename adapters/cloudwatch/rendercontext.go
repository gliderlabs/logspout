package cloudwatch

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"strings"
	"text/template"
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

// HELPER FUNCTIONS

// Searches the OS environment, then the route options, then the render context
// Env for a given key, then uses the value (or the provided default value)
// as template text, which is then rendered in the given context.
// The rendered result is returned - or the default value on any errors.
func (a *Adapter) renderEnvValue(
	envKey string, context *RenderContext, defaultVal string) string {
	finalVal := defaultVal
	if logspoutEnvVal := os.Getenv(envKey); logspoutEnvVal != "" {
		finalVal = logspoutEnvVal // use $envKey, if set
	}
	if routeOptionsVal, exists := a.Route.Options[envKey]; exists {
		finalVal = routeOptionsVal
	}
	if containerEnvVal, exists := context.Env[envKey]; exists {
		finalVal = containerEnvVal // or, $envKey from container!
	}
	template, err := template.New("template").Parse(finalVal)
	if err != nil {
		log.Println("cloudwatch: error parsing template", finalVal, ":", err)
		return defaultVal
	}
	// render the templates in the generated context
	var renderedValue bytes.Buffer
	err = template.Execute(&renderedValue, context)
	if err != nil {
		log.Printf("cloudwatch: error rendering template %s : %s\n",
			finalVal, err)
		return defaultVal
	}
	return renderedValue.String()
}

func parseEnv(envLines []string) map[string]string {
	env := map[string]string{}
	for _, line := range envLines {
		fields := strings.Split(line, `=`)
		if len(fields) > 1 {
			env[fields[0]] = strings.Join(fields[1:], `=`)
		}
	}
	return env
}
