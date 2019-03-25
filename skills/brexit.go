package skills

func BrexitStatus() (string, error) {
	res, err := http.Get("http://r.chipaca.com/howisbrexit.json")
	if err != nil {
		return "", errors.Wrap(err, "getting john's brexit status")
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return "", errors.Errorf("status code error: %d %s", res.StatusCode, res.Status)
	}
	rawBody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", errors.Wrap(err, "reading john's brexit body")
	}
    return string(rawBody), nil

}