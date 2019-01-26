package conply

import "encoding/json"

// Common data marshaller.
func Marshal(data interface{}, indent bool) (string, error) {
	var (
		b []byte
		err error
	)
	if indent {
		b, err = json.MarshalIndent(data, "", "\t")
	} else {
		b, err = json.Marshal(data)
	}
	if err != nil {
		return "", err
	}
	return string(b), err
}

// Common unmarshaller.
func Unmarshal(data string, value interface{}) error {
	err := json.Unmarshal([]byte(data), &value)
	if err != nil {
		return err
	}
	return nil
}

// Marshall data direct to file.
func MarshalFile(path string, data interface{}, indent bool) error {
	value, err := Marshal(data, indent)
	if err != nil {
		return err
	}
	return FilePut(path, value)
}

// Read file contents and unmarshal it.
func UnmarshalFile(path string, value interface{}) error {
	contents, err := FilePull(path)
	if err != nil {
		return err
	}
	err = Unmarshal(contents, &value)
	return err
}
