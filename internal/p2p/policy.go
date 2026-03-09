package p2p

type Policy struct {
	ConfMin          int
	TopicLimit       int
	MsgSizeLimit     int
	RequireSignature bool
	RateLimitPerPeer int
}

func (p Policy) Normalize() Policy {
	if p.ConfMin <= 0 {
		p.ConfMin = 700
	}
	if p.TopicLimit <= 0 {
		p.TopicLimit = 8
	}
	if p.MsgSizeLimit <= 0 {
		p.MsgSizeLimit = 1024
	}
	if p.RateLimitPerPeer <= 0 {
		p.RateLimitPerPeer = 120
	}
	return p
}
