package game

const IntroductionTicks = 258

func IntroductionOpponents(level int, table []byte) [2]string {
	offset := 0x30 + level*8
	return [2]string{string(table[offset : offset+3]), string(table[offset+4 : offset+7])}
}
