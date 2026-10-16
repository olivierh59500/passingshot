// Package sound mixes the extracted samples and music sequences into stereo PCM.
package sound

import (
	"encoding/binary"
	"fmt"
	"sync"

	"github.com/olivierh59500/passingshot/internal/resources"
)

const SampleRate = 44100
const musicRate = 50
const pitchReferenceHz = 3546895

type voice struct {
	pcm                []byte
	position, step     uint64
	volume             int
	loopStart, loopEnd int
	playing            bool
}

type sampleState struct {
	state, duration, volume                                     int16
	basePeriod, bend                                            uint16
	pitchDelay, pitchTicks, pitchAttack, pitchStep, pitchOffset int16
	decay                                                       bool
	instrument                                                  int
}

type channel struct {
	voice                                                               voice
	track, pattern, start                                               int
	repeat, transpose, countdown, instrument, volume                    int
	active                                                              bool
	slideDelay, slideTicks, slideStep, slide                            int
	period, note                                                        int
	patternDone, initialNote                                            bool
	previousNote, noteInstrument                                        int
	localMode, localActive                                              byte
	slideEnabled, slideActive                                           byte
	slidePresetDelay, slidePresetTicks, slidePitch                      byte
	slideRatio                                                          uint32
	legato                                                              byte
	volumeEnabled, volumeActive                                         byte
	volumeInterval, volumePresetCount, volumeIntervalCount, volumeCount byte
	volumeStep, volumeOffset, volumeActiveStep                          int16
	volumeActiveInterval                                                byte
	sample                                                              sampleState
}

// Player is an audio reader. Reading PCM advances only music and sample
// playback; Ebitengine owns gameplay updates independently.
type Player struct {
	mu                                sync.Mutex
	r                                 *resources.Set
	channels                          [4]channel
	effect                            voice
	effectPCM                         []byte
	nextEffect                        int
	song, requestedSong               int
	musicEnabled, status, Muted       bool
	Error                             error
	instrumentMap                     [32]byte
	fadeVolume, fadePeriod, fadeCount int
	frame                             [4]byte
	frameRead, samplesUntilTick       int
}

func New(r *resources.Set) *Player {
	p := &Player{r: r, song: -1, requestedSong: -1, frameRead: 4, nextEffect: -1}
	p.effectPCM = append([]byte(nil), r.Raw["samples"]...)
	for i := range p.effectPCM {
		p.effectPCM[i] ^= 128
	}
	p.selectSong(-1, true)
	return p
}

func (p *Player) SetMuted(muted bool) { p.mu.Lock(); defer p.mu.Unlock(); p.Muted = muted }
func (p *Player) Err() error          { p.mu.Lock(); defer p.mu.Unlock(); return p.Error }

// Song selects music. The pause selection retains the current composition.
func (p *Player) Song(index int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if index == -2 {
		p.musicEnabled = false
		return
	}
	p.requestedSong = -1
	p.selectSong(index, false)
}

func (p *Player) RestartSong(index int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.requestedSong = -1
	p.selectSong(index, true)
}

// RequestSong resumes a retained composition and selects the requested song
// after its final note. Menus can repeat that selection naturally.
func (p *Player) RequestSong(index int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if index < 0 || index >= 5 {
		return
	}
	p.musicEnabled = true
	if index != 0 {
		p.requestedSong = index
	}
	if !p.status {
		p.selectSong(index, true)
	}
}

func (p *Player) selectSong(index int, restart bool) {
	if index == p.song && !restart {
		p.musicEnabled = index >= 0
		return
	}
	p.song = index
	p.musicEnabled = index >= 0 && index < 5
	p.status = p.musicEnabled
	p.channels = [4]channel{}
	p.fadeVolume, p.fadePeriod, p.fadeCount = 64, 0, 1
	if !p.musicEnabled {
		return
	}
	for i := range p.instrumentMap {
		p.instrumentMap[i] = byte(i % 8 * 4)
	}
	tracks := p.r.Raw["music_tracks"]
	for i := range p.channels {
		p.channels[i] = channel{track: int(binary.BigEndian.Uint16(tracks[index*8+i*2:])), pattern: -1, start: -1, repeat: 1, countdown: 1, instrument: -1, volume: 65535, active: true, patternDone: true, initialNote: true, noteInstrument: -2}
	}
}

var effectStarts = [13]int{0, 0x5da, 0xe8b, 0x1684, 0x1e86, 0x2587, 0x2bde, 0x3683, 0x3ccc, 0x4966, 0x545b, 0x6a77, 0x706f}
var effectLengths = [13]int{0x2ed, 0x458, 0x3fc, 0x401, 0x380, 0x32b, 0x552, 0x324, 0x64d, 0x57a, 0xb0e, 0x2fc, 0x6f5}

func (p *Player) Effect(index int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if index < 0 || index >= len(effectStarts) {
		return
	}
	// Game effects suspend the score while retaining its next note.
	p.musicEnabled = false
	p.startEffect(index)
}

func (p *Player) startEffect(index int) {
	if index < 0 || index >= len(effectStarts) {
		return
	}
	start, end := effectStarts[index], effectStarts[index]+effectLengths[index]*2
	if end > len(p.effectPCM) {
		p.Error = fmt.Errorf("effect sample outside resources")
		return
	}
	p.effect = voice{pcm: p.effectPCM[start:end], step: periodStep(1000), volume: 64, playing: true}
	p.nextEffect = -1
	// The double-fault phrase ends with the shared fault sample.
	if index == 9 {
		p.nextEffect = 8
	}
}

func periodStep(period int) uint64 {
	if period < 1 {
		return 0
	}
	return uint64(pitchReferenceHz) << 32 / uint64(period*SampleRate)
}

func (p *Player) Read(dst []byte) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for i := range dst {
		if p.frameRead == 4 {
			p.renderFrame()
		}
		dst[i] = p.frame[p.frameRead]
		p.frameRead++
	}
	return len(dst), nil
}

func (p *Player) renderFrame() {
	if p.samplesUntilTick == 0 {
		p.tick()
		p.samplesUntilTick = SampleRate / musicRate
	}
	p.samplesUntilTick--
	var left, right int
	for i := range p.channels {
		value := p.channels[i].voice.sampleValue()
		if i == 0 || i == 3 {
			left += value
		} else {
			right += value
		}
	}
	if !p.effect.playing && p.nextEffect >= 0 {
		p.startEffect(p.nextEffect)
	}
	value := p.effect.sampleValue()
	left += value / 2
	right += value / 2
	if p.Muted {
		left, right = 0, 0
	}
	binary.LittleEndian.PutUint16(p.frame[:2], uint16(int16(clamp(left, -32768, 32767))))
	binary.LittleEndian.PutUint16(p.frame[2:], uint16(int16(clamp(right, -32768, 32767))))
	p.frameRead = 0
}

func (v *voice) sampleValue() int {
	if !v.playing || len(v.pcm) == 0 {
		return 0
	}
	index := int(v.position >> 32)
	if index >= len(v.pcm) || v.loopEnd > 0 && index >= v.loopEnd {
		if v.loopEnd > v.loopStart {
			length := uint64(v.loopEnd-v.loopStart) << 32
			v.position = (uint64(v.loopStart) << 32) + (v.position-(uint64(v.loopEnd)<<32))%length
			index = int(v.position >> 32)
		} else {
			v.playing = false
			return 0
		}
	}
	value := int(int8(v.pcm[index])) * clamp(v.volume, 0, 64) * 2
	v.position += v.step
	return value
}

func (p *Player) tick() {
	if !p.musicEnabled {
		return
	}
	if !p.status && p.requestedSong >= 0 {
		p.selectSong(p.requestedSong, true)
	}
	p.advanceSamples()
	p.status = false
	for i := range p.channels {
		c := &p.channels[i]
		p.status = p.status || c.active
		if !c.active {
			continue
		}
		for _, step := range []func(*channel) error{p.advanceSlide, p.advanceVolume, p.advance} {
			if err := step(c); err != nil {
				p.Error = err
				c.active = false
				break
			}
		}
	}
}

func (p *Player) triggerInstrument(c *channel, index, note int, duration, volume int16) error {
	ins, err := p.instrument(index)
	if err != nil {
		return err
	}
	ratio, err := p.pitchRatio(int(binary.BigEndian.Uint16(ins[24:])) - int(byte(note)&127))
	if err != nil {
		return err
	}
	period := uint16(uint64(binary.BigEndian.Uint16(ins[26:])) * uint64(ratio) >> 16)
	s := &c.sample
	*s = sampleState{state: 2, duration: duration, volume: min(volume, int16(p.fadeVolume)), basePeriod: period, instrument: index, pitchDelay: int16(binary.BigEndian.Uint16(ins[16:]))}
	if s.pitchDelay >= 0 {
		s.pitchTicks = int16((uint32(binary.BigEndian.Uint16(ins[20:])) + 1) / 2)
		s.pitchAttack = int16(uint32(int32(int16(period))*int32(int16(binary.BigEndian.Uint16(ins[18:])))) >> 15)
		s.pitchStep = int16(uint32(int32(int16(period))*int32(int16(binary.BigEndian.Uint16(ins[22:])))) >> 15)
	}
	data := p.r.Raw["music_instruments"]
	start := int(binary.BigEndian.Uint32(ins))
	end := start + int(binary.BigEndian.Uint16(ins[8:]))*2
	loop := int(binary.BigEndian.Uint32(ins[4:]))
	loopEnd := loop + int(binary.BigEndian.Uint16(ins[10:]))*2
	if start < 0 || end > len(data) {
		return fmt.Errorf("music sample outside resources")
	}
	c.voice = voice{pcm: data[start:end], step: periodStep(int(period)), volume: int(s.volume), playing: true}
	if binary.BigEndian.Uint16(ins[14:]) != 0 && loop >= start && loopEnd <= len(data) && loopEnd > loop {
		c.voice.pcm = data[start:max(end, loopEnd)]
		c.voice.loopStart, c.voice.loopEnd = loop-start, loopEnd-start
	}
	c.period, c.slide = int(period), 0
	return nil
}

func (p *Player) advanceSamples() {
	if p.fadePeriod != 0 {
		p.fadeCount--
		if p.fadeCount <= 0 {
			p.fadeCount = p.fadePeriod
			p.fadeVolume--
		}
	}
	for i := range p.channels {
		c := &p.channels[i]
		s := &c.sample
		if s.state == 0 {
			continue
		}
		ins, err := p.instrument(s.instrument)
		if err != nil {
			p.Error = err
			c.voice.playing = false
			continue
		}
		if s.duration > 0 {
			s.duration--
		}
		if s.pitchDelay != -1 {
			if s.pitchDelay > 0 {
				s.pitchOffset += s.pitchAttack
				s.pitchDelay--
			}
			if s.pitchDelay == 0 {
				s.pitchOffset += s.pitchStep
				s.pitchTicks--
				if s.pitchTicks == 0 && binary.BigEndian.Uint16(ins[20:]) != 0 {
					s.pitchTicks = int16(binary.BigEndian.Uint16(ins[20:]))
					s.pitchStep = -s.pitchStep
				}
			}
		}
		delta := int16(binary.BigEndian.Uint16(ins[14:]))
		if delta != 0 {
			if s.decay {
				s.volume += delta
			} else if s.duration >= 0 && s.duration <= 1 {
				s.decay = true
			}
		} else if s.duration >= 0 && s.duration <= 1 || !c.voice.playing {
			// The score gates a one-shot note even when its recorded sample
			// is longer. Rests and endings must not inherit the sample tail.
			s.volume = 0
		}
		if s.volume <= 0 || p.fadeVolume < 0 {
			s.state = 0
			c.voice.playing = false
			continue
		}
		c.voice.volume = min(int(s.volume), p.fadeVolume)
		c.voice.step = periodStep(int(uint16(s.basePeriod + uint16(s.pitchOffset) + s.bend)))
	}
}

func clamp(v, lo, hi int) int { return min(max(v, lo), hi) }
