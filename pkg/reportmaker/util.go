package reportmaker

import (
	"bytes"
	"encoding/csv"
)

func (rm *reportManager) constructCSVandUpload(bucket, fileName string, data *[][]string) (err error) {
	var buff bytes.Buffer
	w := csv.NewWriter(&buff)
	w.WriteAll(*data)
	err = rm.store.PutObject(bucket, fileName, &buff)
	return
}
