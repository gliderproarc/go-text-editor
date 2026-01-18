package app

import "github.com/gdamore/tcell/v2"

type macroEvent struct {
	Kind      macroEventKind
	Key       tcell.Key
	Rune      rune
	Modifiers tcell.ModMask
}

type macroEventKind int

const (
	macroEventKey macroEventKind = iota
)

type macroStartResult int

const (
	macroStartRecording macroStartResult = iota
	macroStartPrepareRegister
	macroStartDenied
)

func (r *Runner) startMacroRecording(register string) macroStartResult {
	if r.macroPlaying {
		r.updateMacroStatus()
		return macroStartDenied
	}
	if register == "" {
		r.macroPendingRecord = true
		r.updateMacroStatus()
		return macroStartPrepareRegister
	}
	r.macroPendingRecord = false
	r.macroRecording = true
	r.macroRecordRegister = register
	r.macroLastRegister = register
	if r.macroRegisters == nil {
		r.macroRegisters = map[string][]macroEvent{}
	}
	r.macroRegisters[register] = nil
	r.updateMacroStatus()
	return macroStartRecording
}

func (r *Runner) stopMacroRecording() bool {
	if !r.macroRecording {
		r.updateMacroStatus()
		return false
	}
	r.macroRecording = false
	r.macroRecordRegister = ""
	r.macroPendingRecord = false
	r.updateMacroStatus()
	return true
}

func (r *Runner) beginMacroPlayback(register string) bool {
	if r.macroRecording || r.macroPlaying {
		r.updateMacroStatus()
		return false
	}
	r.macroRepeatPending = false
	r.macroRepeatAwaitAt = false
	if register == "" {
		r.macroPendingPlay = true
		r.updateMacroStatus()
		return true
	}
	return r.startMacroPlayback(register)
}

func (r *Runner) startMacroPlayback(register string) bool {
	if register == "" {
		r.updateMacroStatus()
		return false
	}
	r.macroRepeatPending = false
	r.macroRepeatAwaitAt = false
	macro := r.macroRegisters[register]
	if len(macro) == 0 {
		r.updateMacroStatus()
		return false
	}
	r.macroPlaying = true
	r.macroPendingPlay = false
	r.macroPlayback = append([]macroEvent(nil), macro...)
	r.macroLastRegister = register
	r.updateMacroStatus()
	return true
}

func (r *Runner) recordMacroEvent(ev *tcell.EventKey) {
	if !r.shouldRecordMacroEvent(ev) || r.macroRecordRegister == "" {
		return
	}
	r.macroRegisters[r.macroRecordRegister] = append(r.macroRegisters[r.macroRecordRegister], macroEvent{
		Kind:      macroEventKey,
		Key:       ev.Key(),
		Rune:      ev.Rune(),
		Modifiers: ev.Modifiers(),
	})
}

func (r *Runner) macroEventFromKey(ev *tcell.EventKey) macroEvent {
	return macroEvent{Kind: macroEventKey, Key: ev.Key(), Rune: ev.Rune(), Modifiers: ev.Modifiers()}
}

func (r *Runner) consumeMacroEvent() (*tcell.EventKey, bool) {
	if len(r.macroPlayback) == 0 {
		r.macroPlaying = false
		r.macroRepeatPending = true
		r.macroRepeatAwaitAt = false
		r.updateMacroStatus()
		return nil, false
	}
	ev := r.macroPlayback[0]
	r.macroPlayback = r.macroPlayback[1:]
	switch ev.Kind {
	case macroEventKey:
		return tcell.NewEventKey(ev.Key, ev.Rune, ev.Modifiers), true
	default:
		return nil, false
	}
}

func (r *Runner) clearMacroPending() {
	r.macroPendingPlay = false
	r.macroPendingRecord = false
	r.macroRepeatPending = false
	r.updateMacroStatus()
}

func (r *Runner) isMacroCaptureAllowed(ev *tcell.EventKey) bool {
	if !r.macroRecording || r.macroPlaying || r.macroRepeatPending || r.macroRepeatAwaitAt {
		return false
	}
	if r.matchCommand(ev, "save") || r.matchCommand(ev, "quit") || r.matchCommand(ev, "menu") {
		return false
	}
	return true
}

func (r *Runner) shouldRecordMacroEvent(ev *tcell.EventKey) bool {
	if !r.isMacroCaptureAllowed(ev) {
		return false
	}
	if r.macroPendingRecord || r.macroPendingPlay || r.macroRepeatPending || r.macroRepeatAwaitAt {
		return false
	}
	if r.Mode == ModeNormal && ev.Key() == tcell.KeyRune && ev.Modifiers() == 0 {
		switch ev.Rune() {
		case 'q', '@':
			return false
		}
	}
	return true
}

func (r *Runner) handleMacroRepeat(ev *tcell.EventKey) bool {
	if !r.macroRepeatPending {
		return false
	}
	if r.isCancelKey(ev) {
		r.macroRepeatPending = false
		r.macroRepeatAwaitAt = false
		r.updateMacroStatus()
		if r.Screen != nil {
			r.draw(nil)
		}
		return true
	}
	if ev.Key() != tcell.KeyRune || ev.Modifiers() != 0 {
		r.macroRepeatPending = false
		r.macroRepeatAwaitAt = false
		r.updateMacroStatus()
		return false
	}
	if r.macroRepeatAwaitAt {
		r.macroRepeatPending = false
		r.macroRepeatAwaitAt = false
		r.updateMacroStatus()
		return false
	}
	runeKey := ev.Rune()
	if runeKey == '@' {
		r.macroRepeatAwaitAt = true
		return true
	}
	if r.macroRepeatAwaitAt {
		r.macroRepeatAwaitAt = false
		if r.macroLastRegister != "" && string(runeKey) == r.macroLastRegister {
			r.macroRepeatPending = false
			if r.startMacroPlayback(r.macroLastRegister) {
				if r.Screen != nil {
					r.draw(nil)
				}
				return true
			}
			return true
		}
		r.macroRepeatPending = false
		r.updateMacroStatus()
		return false
	}
	if r.macroLastRegister != "" && string(runeKey) == r.macroLastRegister {
		r.macroRepeatPending = false
		if r.startMacroPlayback(r.macroLastRegister) {
			if r.Screen != nil {
				r.draw(nil)
			}
			return true
		}
		return true
	}
	if runeKey == '@' {
		r.macroRepeatPending = false
		if r.macroLastRegister != "" {
			if r.startMacroPlayback(r.macroLastRegister) {
				if r.Screen != nil {
					r.draw(nil)
				}
				return true
			}
			return true
		}
	}
	r.macroRepeatPending = false
	r.updateMacroStatus()
	return false
}

func (r *Runner) handleMacroPending(ev *tcell.EventKey) bool {
	if !r.macroPendingRecord && !r.macroPendingPlay && !r.macroRepeatPending && !r.macroRepeatAwaitAt {
		return false
	}
	if r.isCancelKey(ev) {
		r.clearMacroPending()
		if r.Screen != nil {
			r.draw(nil)
		}
		return true
	}
	if ev.Key() != tcell.KeyRune || ev.Modifiers() != 0 {
		return true
	}
	reg := string(ev.Rune())
	if r.macroPendingRecord {
		r.macroPendingRecord = false
		r.startMacroRecording(reg)
		if r.Screen != nil {
			r.draw(nil)
		}
		return true
	}
	if r.macroPendingPlay {
		r.macroPendingPlay = false
		r.startMacroPlayback(reg)
		if r.Screen != nil {
			r.draw(nil)
		}
		return true
	}
	if r.macroRepeatPending {
		r.macroRepeatPending = false
		r.macroRepeatAwaitAt = false
		if r.macroLastRegister != "" && reg == r.macroLastRegister {
			if r.startMacroPlayback(reg) {
				if r.Screen != nil {
					r.draw(nil)
				}
				return true
			}
		}
		r.updateMacroStatus()
		if r.Screen != nil {
			r.draw(nil)
		}
		return false
	}
	r.updateMacroStatus()
	if r.Screen != nil {
		r.draw(nil)
	}
	return false
}

func (r *Runner) maybeMacroMenuAction(action func() bool) func() bool {
	return func() bool {
		r.macroPendingRecord = false
		r.macroPendingPlay = false
		r.macroRepeatPending = false
		r.macroRepeatAwaitAt = false
		r.updateMacroStatus()
		return action()

	}
}

func (r *Runner) macroStatusLine() string {
	if r.macroPlaying {
		if r.macroLastRegister != "" {
			return "Playing macro @" + r.macroLastRegister
		}
		return "Playing macro"
	}
	if r.macroRepeatPending {
		if r.macroLastRegister != "" {
			return "Replay macro @" + r.macroLastRegister + " or @@"
		}
		return "Replay macro"
	}
	if r.macroRepeatAwaitAt {
		if r.macroLastRegister != "" {
			return "Replay macro @@" + r.macroLastRegister
		}
		return "Replay macro @@"
	}
	if r.macroRepeatPending {
		if r.macroLastRegister != "" {
			return "Replay macro @" + r.macroLastRegister
		}
		return "Replay macro"
	}
	if r.macroRecording {
		if r.macroRecordRegister != "" {
			return "Recording macro @" + r.macroRecordRegister
		}
		return "Recording macro"
	}
	if r.macroPendingRecord {
		return "Macro record: choose register"
	}
	if r.macroPendingPlay {
		return "Macro play: choose register"
	}
	return ""
}

func (r *Runner) updateMacroStatus() {
	r.MacroStatus = r.macroStatusLine()
}

func (r *Runner) lastMacroRegister() string {
	return r.macroLastRegister
}
