package k8s

import "encoding/json"

func DecodeMap(in map[string]any, to any) error {

	bs, err := json.Marshal(in)

	if err != nil {
		return err
	}

	return json.Unmarshal(bs, to)

}

func EncodeToMap(in any) (map[string]any, error) {

	bs, err := json.Marshal(in)

	if err != nil {
		return nil, err
	}

	to := map[string]any{}

	err = json.Unmarshal(bs, &to)

	if err != nil {
		return nil, err
	}

	return to, nil
}
