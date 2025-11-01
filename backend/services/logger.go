package services

import (
	"context"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type Livello string

const (
	Info  Livello = "info"
	Warn  Livello = "warn"
	Error Livello = "error"
)

type Logger struct {
	ctx      context.Context
	running  bool
	stopChan chan struct{}
}

// Deve essere chiamato in OnStartup per avere ctx
func (l *Logger) SetContext(ctx context.Context) {
	l.ctx = ctx
}

func (l *Logger) StartDemoLogs() {
	if l.running || l.ctx == nil {
		return
	}
	l.running = true
	l.stopChan = make(chan struct{})
	go func() {
		t := time.NewTicker(2 * time.Second)
		defer t.Stop()
		i := 0
		for {
			select {
			case <-l.stopChan:
				return
			case <-t.C:
				i++
				l.Emit(Info, "Log di prova n."+time.Now().Format("15:04:05"))
				if i%5 == 0 {
					l.Emit(Warn, "Attenzione demo: controlla il carico SNMP")
				}
				if i%11 == 0 {
					l.Emit(Error, "Errore demo: timeout richiesta SNMP")
				}
			}
		}
	}()
}

func (l *Logger) StopDemoLogs() {
	if !l.running {
		return
	}
	close(l.stopChan)
	l.running = false
}

func (l *Logger) Emit(level Livello, msg string) {
	if l.ctx == nil {
		return
	}
	payload := map[string]any{
		"livello":    level,
		"messaggio":  msg,
		"timestamp":  time.Now().Format(time.RFC3339),
	}
	runtime.EventsEmit(l.ctx, "log:event", payload)
}
