package cefevent

import (
	"errors"
	"fmt"
	"log"
	"os"
	"reflect"
	"sort"
	"strings"
)

type CefEventer interface {
	Generate() (string, error)
	Validate() bool
	Log() (bool, error)
}

type CefEvent struct {
	// defaults to 0 which is also the first CEF version.
	Version            int
	DeviceVendor       string
	DeviceProduct      string
	DeviceVersion      string
	DeviceEventClassId string
	Name               string
	Severity           string
	Extensions         map[string]string
}

func cefEscapeField(field string) string {

	replacer := strings.NewReplacer(
		"\\", "\\\\",
		"|", "\\|",
		"\n", "\\n",
	)

	return replacer.Replace(field)
}

func cefEscapeExtension(field string) string {

	replacer := strings.NewReplacer(
		"\\", "\\\\", "\n",
		"\\n", "=", "\\=",
	)

	return replacer.Replace(field)
}

func (event *CefEvent) Validate() bool {

	assertEvent := reflect.ValueOf(event).Elem()

	// define an array with all the mandatory
	// CEF fields.
	mandatoryFields := []string{
		"Version",
		"DeviceVendor",
		"DeviceProduct",
		"DeviceVersion",
		"DeviceEventClassId",
		"Name",
		"Severity",
	}

	// loop over all mandatory fields
	// and verify if they are not empty
	// according to their String type.
	for _, field := range mandatoryFields {

		if assertEvent.FieldByName(field).String() == "" {
			return false
		}
	}

	return true

}

// Log should be used as a stub in most cases, it either
// succeeds generating the CEF event and send it to stdout
// or doesnt and logs that to stderr. This function
// plays well inside containers.
func (event *CefEvent) Log() (bool, error) {

	logMessage, err := event.Generate()

	if err != nil {
		log.SetOutput(os.Stderr)
		errMsg := "Unable to generate and thereby log the CEF message."
		log.Println(errMsg)
		return false, errors.New(errMsg)
	}

	log.SetOutput(os.Stdout)
	log.Println(logMessage)
	return true, nil
}

func (event CefEvent) Generate() (string, error) {

	if !CefEventer.Validate(&event) {
		return "", errors.New("Not all mandatory CEF fields are set.")
	}

	event.DeviceVendor = cefEscapeField(event.DeviceVendor)
	event.DeviceProduct = cefEscapeField(event.DeviceProduct)
	event.DeviceVersion = cefEscapeField(event.DeviceVersion)
	event.DeviceEventClassId = cefEscapeField(event.DeviceEventClassId)
	event.Name = cefEscapeField(event.Name)
	event.Severity = cefEscapeField(event.Severity)

	var p strings.Builder

	var sortedExtensions []string
	for k := range event.Extensions {
		sortedExtensions = append(sortedExtensions, k)
	}
	sort.Strings(sortedExtensions)

	// construct the extension string according to the CEF format
	for _, k := range sortedExtensions {
		p.WriteString(fmt.Sprintf(
			"%s=%s ",
			cefEscapeExtension(k),
			cefEscapeExtension(event.Extensions[k])),
		)
	}

	// make sure there is not a trailing space for the extension
	// fields according to the CEF standard.
	extensionString := strings.TrimSpace(p.String())

	eventCef := fmt.Sprintf(
		"CEF:%v|%v|%v|%v|%v|%v|%v|%v",
		event.Version, event.DeviceVendor,
		event.DeviceProduct, event.DeviceVersion,
		event.DeviceEventClassId, event.Name,
		event.Severity, extensionString,
	)

	return eventCef, nil
}
