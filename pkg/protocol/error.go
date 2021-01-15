package protocol

type Error struct {
	Description string `json:"description"`
	Code        int32  `json:"code"`
}
