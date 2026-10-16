package game

func NewServiceBoard(random uint32) [4]byte {
	loser := byte('1' + ((random >> 16) & 3))
	if random&0x800 != 0 {
		return [4]byte{'1', '6', '6', loser}
	}
	return [4]byte{'6', loser, '1', '6'}
}
