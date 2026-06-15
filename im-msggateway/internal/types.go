package internal

import "sync"

type Request struct {
	ReqIdentifier int    `json:"reqIdentifier"`
	Token         string `json:"token"`
	SenderID      string `json:"senderID"`
	OperationID   string `json:"operationID"`
	MsgIncr       string `json:"msgIncr"`
	Data          []byte `json:"data"`
}

func (r *Request) Reset() {
	r.ReqIdentifier = 0
	r.Token = ""
	r.SenderID = ""
	r.OperationID = ""
	r.MsgIncr = ""
	r.Data = nil
}

var reqPool = sync.Pool{
	New: func() interface{} {
		return &Request{}
	},
}

func MallocRequest() *Request {
	req := reqPool.Get().(*Request)
	req.Reset()
	return req
}

func FreeRequest(req *Request) {
	reqPool.Put(req)
}

type Response struct {
	ReqIdentifier int    `json:"reqIdentifier"`
	MsgIncr       string `json:"msgIncr"`
	OperationID   string `json:"operationID"`
	ErrCode       int    `json:"errCode"`
	ErrMsg        string `json:"errMsg"`
	Data          []byte `json:"data"`
}
