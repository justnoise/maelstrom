package messages

// "{"id":0,"src":"c0","dest":"n1","body":{"type":"init","node_id":"n1","node_ids":["n1"],"msg_id":1}}\n"
type Request struct {
	ID   int         `json:"id"`
	Src  string      `json:"src"`
	Dest string      `json:"dest"`
	Body RequestBody `json:"body"`
}

// "type":"init","node_id":"n1","node_ids":["n1"],"msg_id":1
type RequestBody struct {
	Type      string              `json:"type"`
	MsgID     int                 `json:"msg_id"`
	InReplyTo int                 `json:"in_reply_to"`
	NodeID    string              `json:"node_id,omitempty"`
	NodeIDs   []string            `json:"node_ids,omitempty"`
	Echo      string              `json:"echo,omitempty"`
	Topology  map[string][]string `json:"topology,omitempty"`
	Message   interface{}         `json:"message,omitempty"`
}

type ReplyBodyable interface {
	SetMsgID(int)
	SetInReplyTo(int)
}

type Reply struct {
	Src  string        `json:"src"`
	Dest string        `json:"dest"`
	Body ReplyBodyable `json:"body"`
}

type ReplyBodyBase struct {
	MsgID     int    `json:"msg_id"`
	InReplyTo int    `json:"in_reply_to"`
	Type      string `json:"type"`
}

func (r *ReplyBodyBase) SetMsgID(id int) {
	r.MsgID = id
}

func (r *ReplyBodyBase) SetInReplyTo(id int) {
	r.InReplyTo = id
}

type ReplyBodyEcho struct {
	ReplyBodyBase
	Echo string `json:"echo"`
}

type ReplyBodyMessages struct {
	ReplyBodyBase
	Messages []interface{} `json:"messages"`
}
