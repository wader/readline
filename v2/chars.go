package readline

const (
	CharCtrlA     = 0x01
	CharLineStart = CharCtrlA

	CharCtrlB    = 0x02
	CharBackward = CharCtrlB

	CharCtrlC     = 0x03
	CharInterrupt = CharCtrlC

	CharCtrlD  = 0x04
	CharDelete = CharCtrlD

	CharCtrlE   = 0x05
	CharLineEnd = CharCtrlE

	CharCtrlF   = 0x06
	CharForward = CharCtrlF

	CharCtrlG = 0x07
	CharBell  = CharCtrlG

	CharCtrlH = 0x08

	CharCtrlI = 0x09
	CharTab   = CharCtrlI

	CharCtrlJ = 0x0A

	CharCtrlK = 0x0B
	CharKill  = CharCtrlK

	CharCtrlL = 0x0C
	CharClear = CharCtrlL

	CharCtrlM = 0x0D
	CharEnter = CharCtrlM

	CharCtrlN = 0x0E
	CharNext  = CharCtrlN

	CharCtrlO = 0x0F

	CharCtrlP = 0x10
	CharPrev  = CharCtrlP

	CharCtrlQ = 0x11

	CharCtrlR     = 0x12
	CharBckSearch = CharCtrlR

	CharCtrlS     = 0x13
	CharFwdSearch = CharCtrlS

	CharCtrlT     = 0x14
	CharTranspose = CharCtrlT

	CharCtrlU     = 0x15
	CharKillFront = CharCtrlU

	CharCtrlV = 0x16

	CharCtrlW = 0x17

	CharCtrlX = 0x18

	CharCtrlY = 0x19
	CharYank  = CharCtrlY

	CharCtrlZ   = 0x1A
	CharSuspend = CharCtrlZ

	CharEscape = 0x1B

	CharBackspace = 0x7F
)
