package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"mib-to-the-future/backend/mib"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App è la struttura principale dell'applicazione.
type App struct {
	ctx           context.Context
	mibDB         *mib.Database
	mibInitErr    error
	oidNameCache  map[string]string
	oidBaseCache  map[string]string
	oidNodeCache  map[string]*mib.Node
	oidNameCacheM sync.RWMutex
}

// NewApp crea una nuova istanza dell'applicazione.
func NewApp() *App {
	return &App{
		oidNameCache: make(map[string]string),
		oidBaseCache: make(map[string]string),
		oidNodeCache: make(map[string]*mib.Node),
	}
}

// mibNotInitializedErr restituisce un errore appropriato se il database MIB non è inizializzato.
func (a *App) mibNotInitializedErr() error {
	if a == nil {
		return fmt.Errorf("MIB database not initialized")
	}
	if a.mibInitErr != nil {
		return fmt.Errorf("MIB database not initialized: %v", a.mibInitErr)
	}
	return fmt.Errorf("MIB database not initialized")
}

// Startup inizializza l'applicazione al momento dell'avvio.
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx

	if a.oidNameCache == nil {
		a.oidNameCache = make(map[string]string)
	}
	if a.oidBaseCache == nil {
		a.oidBaseCache = make(map[string]string)
	}
	if a.oidNodeCache == nil {
		a.oidNodeCache = make(map[string]*mib.Node)
	}

	// Ottieni la directory di configurazione standard per l'OS corrente
	configDir, err := os.UserConfigDir()
	if err != nil {
		a.mibInitErr = fmt.Errorf("failed to resolve user config dir: %w", err)
		runtime.LogError(ctx, a.mibInitErr.Error())
		return
	}

	// Crea il path per i dati della nostra app
	dataDir := filepath.Join(configDir, "MIB to the Future")

	// Inizializza database MIB
	a.mibDB, err = mib.NewDatabase(dataDir)
	if err != nil {
		a.mibInitErr = fmt.Errorf("failed to initialize MIB database in %s: %w", dataDir, err)
		runtime.LogError(ctx, a.mibInitErr.Error())
		return
	}
	a.mibInitErr = nil

	// Esegui migrazioni del database
	if err := a.runMigrations(); err != nil {
		a.mibInitErr = fmt.Errorf("database migration failed: %w", err)
		runtime.LogError(ctx, a.mibInitErr.Error())
		return
	}

	// Precarica i MIB standard comuni all'avvio per evitare errori di dipendenze mancanti
	runtime.LogInfo(ctx, "Preloading standard MIB modules...")
	parser := mib.NewParser(a.mibDB)
	if err := parser.PreloadStandardMIBs(dataDir); err != nil {
		// Non è un errore fatale, logga e continua
		runtime.LogWarning(ctx, fmt.Sprintf("Failed to preload some standard MIBs: %v", err))
	} else {
		runtime.LogInfo(ctx, "Standard MIBs preloaded successfully")
	}

	runtime.LogInfo(ctx, fmt.Sprintf("MIB database ready at: %s", dataDir))
}

// runMigrations esegue le migrazioni del database.
func (a *App) runMigrations() error {
	if a.mibDB == nil {
		return fmt.Errorf("database not initialized")
	}

	return a.mibDB.EnsureHostConfigSchema()
}

// shutdown chiude l'applicazione.
func (a *App) shutdown(ctx context.Context) {
	if a.mibDB != nil {
		a.mibDB.Close()
	}
}

// Greet restituisce un saluto personalizzato.
// È una funzione di esempio per dimostrare il binding tra Go e il frontend.
func (a *App) Greet(nome string) string {
	if nome == "" {
		nome = "MIBnauta"
	}
	return "Ciao " + nome + ", benvenuto su MIB to the Future!"
}
