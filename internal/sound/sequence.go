package sound

import (
	"encoding/binary"
	"fmt"
	"math/bits"
)

func (p *Player) advance(c *channel) error {
	patterns, tracks := p.r.Raw["music_patterns"], p.r.Raw["music_tracks"]
	channelIndex := 0
	for i := range p.channels {
		if c == &p.channels[i] {
			channelIndex = i
			break
		}
	}
	byteAt := func(data []byte, cursor *int) (byte, error) {
		if *cursor < 0 || *cursor >= len(data) {
			return 0, fmt.Errorf("music cursor outside resource")
		}
		v := data[*cursor]
		*cursor++
		return v, nil
	}
	for operations := 0; operations < 128; operations++ {
		if c.patternDone {

			c.repeat = int(byte(c.repeat - 1))
			if c.repeat != 0 {

				c.pattern, c.patternDone = c.start, false
			} else {

				c.repeat = 1
				cursor := c.track
				selected := false
				for commands := 0; commands < 128; commands++ {
					cmd, err := byteAt(tracks, &cursor)
					if err != nil {
						return err
					}

					switch {
					case cmd == 0xff:
						if c.sample.state != 0 && uint16(c.sample.state) != 0x8000 {
							return nil
						}
						c.active = false
						return nil
					case cmd == 0xfe:
						v, err := byteAt(tracks, &cursor)
						if err != nil {
							return err
						}
						c.transpose = int(int8(v))
					case cmd == 0xfd:
						v, err := byteAt(tracks, &cursor)
						if err != nil {
							return err
						}
						p.fadePeriod = int(int16(int8(v)))
					case cmd >= 0xf0:
						v, err := byteAt(tracks, &cursor)
						if err != nil {
							return err
						}
						index := channelIndex*8 + int(cmd-0xf0)
						if index >= len(p.instrumentMap) {
							return fmt.Errorf("music instrument map outside resource")
						}
						p.instrumentMap[index] = v
					case cmd >= 0xc8:
						c.repeat = int(cmd - 0xc8)
					default:
						at := 0x460 + int(cmd)*2
						if at+2 > len(patterns) {
							return fmt.Errorf("unknown music pattern %d", cmd)
						}
						c.start = int(int16(binary.BigEndian.Uint16(patterns[at:])))
						c.pattern, c.patternDone, c.track = c.start, false, cursor
						selected = true
					}
					if selected {
						break
					}
				}
				if !selected {
					return fmt.Errorf("music track command limit exceeded")
				}
			}
		} else {

		}

		c.countdown = int(byte(c.countdown - 1))
		if c.pattern < 0 || c.pattern >= 0x460 {
			return fmt.Errorf("music pattern outside resource")
		}
		if c.countdown != 0 {

			if patterns[c.pattern]&0x80 != 0 {

				if err := p.patternCommand(c, channelIndex); err != nil {
					return err
				}
			} else {

			}
			return nil
		}

		for commands := 0; commands < 128; commands++ {
			if c.pattern < 0 || c.pattern >= 0x460 {
				return fmt.Errorf("music pattern outside resource")
			}
			if patterns[c.pattern]&0x80 == 0 {

				break
			}

			if err := p.patternCommand(c, channelIndex); err != nil {
				return err
			}
			if c.patternDone {

				break
			}

		}
		if c.patternDone {

			if c.countdown == 0 {
				c.countdown = 1

			}

			continue
		}
		cmd, err := byteAt(patterns, &c.pattern)
		if err != nil {
			return err
		}
		if cmd == 0x7f {

			duration, err := byteAt(patterns, &c.pattern)
			if err != nil {
				return err
			}
			c.countdown = int(duration)
			return nil
		}

		c.note = int(byte(int(cmd) + c.transpose))
		c.sample.bend = 0
		c.localActive = c.localMode | 0xc0
		c.slideActive, c.slideDelay, c.slideTicks = c.slideEnabled, int(c.slidePresetDelay), int(c.slidePresetTicks)
		if c.slideActive != 0 {
			ratio, err := p.pitchRatio(-int(int8(c.slidePitch)))
			if err != nil {
				return err
			}
			c.slideRatio = ratio
		}
		c.volumeActive = c.volumeEnabled
		if c.volumeActive != 0 {
			c.volumeIntervalCount, c.volumeCount = c.volumeInterval, c.volumePresetCount
			c.volumeActiveInterval, c.volumeActiveStep = c.volumeInterval, c.volumeStep
			c.volumeOffset = 0
		}
		duration, err := byteAt(patterns, &c.pattern)
		if err != nil {
			return err
		}
		sampleDuration := int16(duration)
		if duration == 0 {

			duration, err = byteAt(patterns, &c.pattern)
			if err != nil {
				return err
			}
			sampleDuration = -1
		} else {

		}

		c.countdown = int(duration)
		if c.legato != 0 {
			c.slideActive, c.slideDelay, c.slideTicks = 255, 0, int(c.slidePresetTicks)
			next := c.note
			if !c.initialNote {
				c.note = c.previousNote
			}
			difference := int(int8(byte(next - c.note)))
			ratio, err := p.pitchRatio(-difference)
			if err != nil {
				return err
			}
			c.slideRatio = ratio
		}
		c.previousNote, c.noteInstrument = c.note, c.instrument
		volume, err := p.noteVolume(c)
		if err != nil {
			return err
		}

		if err = p.triggerInstrument(c, c.instrument, c.note, sampleDuration, volume); err != nil {
			return err
		}
		c.initialNote = false
		return nil
	}
	return fmt.Errorf("music pattern command limit exceeded")
}

func (p *Player) patternCommand(c *channel, channelIndex int) error {
	data := p.r.Raw["music_patterns"]
	read := func() (byte, error) {
		if c.pattern < 0 || c.pattern >= 0x460 {
			return 0, fmt.Errorf("music command outside resource")
		}
		v := data[c.pattern]
		c.pattern++
		return v, nil
	}
	cmd, err := read()
	if err != nil {
		return err
	}

	switch {
	case cmd <= 0x88:
		index := channelIndex*8 + int(cmd&7)
		mapped := int(p.instrumentMap[index])
		if mapped&3 != 0 {
			return fmt.Errorf("unaligned music instrument map")
		}
		c.instrument = mapped / 4
	case cmd == 0xff:
		c.patternDone = true
	case cmd < 0xc0:
		c.localMode = cmd & 15
	case cmd == 0xc0:
		c.slideEnabled, c.legato = 255, 0
		c.slidePresetDelay, err = read()
		if err != nil {
			return err
		}
		c.slidePresetTicks, err = read()
		if err != nil {
			return err
		}
		c.slidePitch, err = read()
		return err
	case cmd == 0xc1:
		c.slideEnabled = 0
	case cmd == 0xc2:
		c.volumeEnabled = 255
		c.volumeInterval, err = read()
		if err != nil {
			return err
		}
		c.volumePresetCount, err = read()
		if err != nil {
			return err
		}
		step, err := read()
		if err != nil {
			return err
		}
		c.volumeStep = int16(int8(step))
	case cmd == 0xc3:
		c.volumeEnabled = 0
	case cmd == 0xc4:
		c.legato, c.slideEnabled = 255, 0
		c.slidePresetTicks, err = read()
		return err
	case cmd == 0xc5:
		c.legato = 0
	case cmd == 0xc6:
		value, err := read()
		if err != nil {
			return err
		}
		c.volume = int(uint16(value)<<8 | uint16(c.volume)&255)
	default:
		return fmt.Errorf("unknown music command %#x", cmd)
	}
	return nil
}

func (p *Player) instrument(index int) ([]byte, error) {
	data := p.r.Raw["music_instruments"]
	if index < 0 || index >= 6 {
		return nil, fmt.Errorf("unknown music instrument %d", index)
	}
	offset := int(binary.BigEndian.Uint32(data[index*4:]))
	if offset < 0 || offset+28 > len(data) {
		return nil, fmt.Errorf("music instrument outside resource")
	}
	return data[offset : offset+28], nil
}

func (p *Player) pitchRatio(semitones int) (uint32, error) {
	index := 48 + semitones
	data := p.r.Raw["music_pitch"]
	if index < 0 || index*4+4 > len(data) {
		return 0, fmt.Errorf("music pitch outside original table")
	}
	return binary.BigEndian.Uint32(data[index*4:]), nil
}

func (p *Player) noteVolume(c *channel) (int16, error) {
	ins, err := p.instrument(c.instrument)
	if err != nil {
		return 0, err
	}
	product := uint32(binary.BigEndian.Uint16(ins[12:])) * uint32(uint16(c.volume))

	return int16(bits.RotateLeft32(product, 18)), nil
}

func (p *Player) trigger(c *channel) error {
	c.noteInstrument = c.instrument
	c.sample.bend = 0
	volume, err := p.noteVolume(c)
	if err != nil {
		return err
	}
	return p.triggerInstrument(c, c.instrument, c.note, int16(c.countdown), volume)
}

func (p *Player) advanceSlide(c *channel) error {
	if c.slideActive == 0 {
		return nil
	}
	if int8(c.slideActive) < 0 {
		if c.slideTicks == 0 {
			return fmt.Errorf("zero-length music slide")
		}
		base := c.sample.basePeriod
		target := uint16(uint64(base) * uint64(c.slideRatio) >> 16)
		delta := int16(target - base)
		step := int16(int32(delta) / int32(byte(c.slideTicks)))
		c.slideStep = int(step)
		c.slideRatio = uint32(uint16(step))<<16 | c.slideRatio&65535
		c.slideActive &= 127
	}
	if c.slideDelay != 0 {
		c.slideDelay = int(byte(c.slideDelay - 1))
		return nil
	}
	if c.slideTicks != 0 {
		c.slideTicks = int(byte(c.slideTicks - 1))
		c.sample.bend += uint16(c.slideStep)
		c.slide = int(int16(c.sample.bend))
	}
	return nil
}

func (p *Player) advanceVolume(c *channel) error {
	if c.volumeActive == 0 || c.volumeCount == 0 {
		return nil
	}
	c.volumeIntervalCount--
	if c.volumeIntervalCount != 0 {
		return nil
	}
	c.volumeCount--
	c.volumeIntervalCount = c.volumeActiveInterval
	ins, err := p.instrument(c.noteInstrument)
	if err != nil {
		return err
	}
	c.volumeOffset += c.volumeActiveStep
	target := int16(binary.BigEndian.Uint16(ins[12:])) + c.volumeOffset
	if target < 0 {
		c.volumeCount = 0
		return nil
	}
	if target > 64 {
		target = 64
	}
	return p.triggerInstrument(c, c.noteInstrument, c.note, int16(c.volumeActiveInterval), target)
}
