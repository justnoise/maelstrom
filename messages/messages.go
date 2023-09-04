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
	Type     string              `json:"type"`
	MsgID    int                 `json:"msg_id"`
	NodeID   string              `json:"node_id,omitempty"`
	NodeIDs  []string            `json:"node_ids,omitempty"`
	Echo     string              `json:"echo,omitempty"`
	Topology map[string][]string `json:"topology,omitempty"`
	Message  interface{}         `json:"message,omitempty"`
}

func (r *RequestBody) SetMsgID(id int) {
	r.MsgID = id
}

type Reply struct {
	Src  string        `json:"src"`
	Dest string        `json:"dest"`
	Body ReplyBodyable `json:"body"`
}

type Bodyable interface {
	SetMsgID(int)
}

type ReplyBodyable interface {
	Bodyable
	SetInReplyTo(int)
}

type ReplyBase struct {
	MsgID     int    `json:"msg_id"`
	InReplyTo int    `json:"in_reply_to"`
	Type      string `json:"type"`
}

func (r *ReplyBase) SetMsgID(id int) {
	r.MsgID = id
}

func (r *ReplyBase) SetInReplyTo(id int) {
	r.InReplyTo = id
}

type ReplyEcho struct {
	ReplyBase
	Echo string `json:"echo"`
}

type ReplyMessages struct {
	ReplyBase
	Messages []interface{} `json:"messages"`
}

// type ReplyBody struct {
// 	MsgID     int      `json:"msg_id"`
// 	InReplyTo int      `json:"in_reply_to"`
// 	Type      string   `json:"type"`
// 	Echo      string   `json:"echo,omitempty"`
// 	Messages  []string `json:"messages,omitempty"`
// }
