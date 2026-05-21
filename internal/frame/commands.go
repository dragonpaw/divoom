package frame

import "context"

// Channel/GetClockInfo — returns the dial currently displayed and the screen
// brightness. Useful as a probe / health check.
type getClockInfoReq struct {
	Command string `json:"Command"`
}

type ClockInfo struct {
	ClockID    int `json:"ClockId"`
	Brightness int `json:"Brightness"`
}

func (c *Client) GetClockInfo(ctx context.Context) (ClockInfo, error) {
	var out ClockInfo
	err := c.Call(ctx, getClockInfoReq{Command: "Channel/GetClockInfo"}, &out)
	return out, err
}

// Device/EnterCustomControlMode — installs a custom layout: one 800x1280
// background plus a DispList of Text / Image / NetData / built-in elements.
// See https://docin.divoom-gz.com/web/#/5/374 for element-type details.
type CustomMode struct {
	Command                 string    `json:"Command"`
	BackgroundImageLocalFlag int      `json:"BackgroudImageLocalFlag"` // sic: Divoom drops the 'n'
	BackgroundImageAddr      string   `json:"BackgroudImageAddr"`      // sic
	DispList                 []DispElement `json:"DispList"`
}

// DispElement is a single positioned element inside a custom layout. Not
// every field applies to every Type — see the docs for the matrix.
type DispElement struct {
	ID           int    `json:"ID"`
	Type         string `json:"Type"` // Text | Image | NetData | Time | Date | MonYear | Mday | Year | Month | Week | Weather | Temperature
	StartX       int    `json:"StartX"`
	StartY       int    `json:"StartY"`
	Width        int    `json:"Width"`
	Height       int    `json:"Height"`
	Align        int    `json:"Align"` // 0=left, 1=right, 2=middle
	FontSize     int    `json:"FontSize,omitempty"`
	FontID       int    `json:"FontID,omitempty"`
	FontColor    string `json:"FontColor,omitempty"`
	BgColor      string `json:"BgColor,omitempty"`
	Url          string `json:"Url,omitempty"`
	RuleInfo     string `json:"RuleInfo,omitempty"`
	TextMessage  string `json:"TextMessage,omitempty"`
	RequestTime  int    `json:"RequestTime,omitempty"` // seconds; min 10 for NetData
	ImgLocalFlag int    `json:"ImgLocalFlag,omitempty"` // 1 = /userdata/... local file
}

func (c *Client) EnterCustomMode(ctx context.Context, m CustomMode) error {
	m.Command = "Device/EnterCustomControlMode"
	return c.Call(ctx, m, nil)
}

// Device/ExitCustomControlMode — restore the previously-selected preset dial.
type exitCustomReq struct {
	Command string `json:"Command"`
}

func (c *Client) ExitCustomMode(ctx context.Context) error {
	return c.Call(ctx, exitCustomReq{Command: "Device/ExitCustomControlMode"}, nil)
}

// Device/UpdateDisplayItems — patch the TextMessage of one or more existing
// Text elements without re-sending the whole layout. Image elements cannot
// be patched this way per the docs.
type updateDisplayReq struct {
	Command  string             `json:"Command"`
	DispList []TextUpdate       `json:"DispList"`
}

type TextUpdate struct {
	ID          int    `json:"ID"`
	TextMessage string `json:"TextMessage"`
}

func (c *Client) UpdateTexts(ctx context.Context, updates []TextUpdate) error {
	return c.Call(ctx, updateDisplayReq{
		Command:  "Device/UpdateDisplayItems",
		DispList: updates,
	}, nil)
}

// Channel/SetClockSelectId — switch to a preset dial by ClockId.
type setClockReq struct {
	Command string `json:"Command"`
	ClockID int    `json:"ClockId"`
}

func (c *Client) SelectClock(ctx context.Context, clockID int) error {
	return c.Call(ctx, setClockReq{Command: "Channel/SetClockSelectId", ClockID: clockID}, nil)
}
