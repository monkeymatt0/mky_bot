package worker

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/robfig/cron/v3"
)

// CronWorker definisce l'interfaccia standard per tutti i worker cron
type CronWorker interface {
	// ExecuteTradingCycle esegue un ciclo di lavoro del worker
	ExecuteTradingCycle()

	// Stop ferma il worker e pulisce le risorse
	Stop()

	// GetName restituisce il nome del worker per identificazione
	GetName() string
}

// WorkerConfig contiene la configurazione per un worker
type WorkerConfig struct {
	Name        string     // Nome identificativo del worker
	Schedule    string     // Cron schedule (es: "* * * * *" per ogni minuto)
	Worker      CronWorker // Istanza del worker
	Enabled     bool       // Se il worker è abilitato
	Description string     // Descrizione del worker
}

// WorkerManager gestisce tutti i worker con cron scheduling
type WorkerManager struct {
	cron      *cron.Cron
	workers   map[string]*WorkerConfig
	ctx       context.Context
	cancel    context.CancelFunc
	mutex     sync.RWMutex
	isRunning bool
}

// NewWorkerManager crea una nuova istanza di WorkerManager
func NewWorkerManager() *WorkerManager {
	ctx, cancel := context.WithCancel(context.Background())

	// Crea cron con logging personalizzato
	cronLogger := cron.VerbosePrintfLogger(log.New(os.Stdout, "CRON: ", log.LstdFlags))

	return &WorkerManager{
		cron:    cron.New(cron.WithLogger(cronLogger), cron.WithSeconds()),
		workers: make(map[string]*WorkerConfig),
		ctx:     ctx,
		cancel:  cancel,
	}
}

// RegisterWorker registra un nuovo worker con la sua schedulazione
func (wm *WorkerManager) RegisterWorker(config *WorkerConfig) error {
	wm.mutex.Lock()
	defer wm.mutex.Unlock()

	if _, exists := wm.workers[config.Name]; exists {
		return fmt.Errorf("worker %s già registrato", config.Name)
	}

	if !config.Enabled {
		log.Printf("⚠️  Worker %s registrato ma DISABILITATO", config.Name)
		wm.workers[config.Name] = config
		return nil
	}

	// Wrapper per il job che gestisce errori e context
	jobWrapper := func() {
		select {
		case <-wm.ctx.Done():
			log.Printf("🛑 Worker %s: Context cancellato, salto esecuzione", config.Name)
			return
		default:
		}

		log.Printf("🚀 Worker %s: Inizio esecuzione ciclo", config.Name)
		start := time.Now()

		// Recupera panic per evitare crash del cron
		defer func() {
			if r := recover(); r != nil {
				log.Printf("❌ Worker %s: PANIC recuperato: %v", config.Name, r)
			}
		}()

		// Esegui il worker
		config.Worker.ExecuteTradingCycle()

		duration := time.Since(start)
		log.Printf("✅ Worker %s: Ciclo completato in %v", config.Name, duration)
	}

	// Aggiungi il job al cron
	entryID, err := wm.cron.AddFunc(config.Schedule, jobWrapper)
	if err != nil {
		return fmt.Errorf("errore aggiunta job cron per worker %s: %w", config.Name, err)
	}

	wm.workers[config.Name] = config
	log.Printf("✅ Worker %s registrato con schedule '%s' (Entry ID: %d)",
		config.Name, config.Schedule, entryID)

	return nil
}

// RemoveWorker rimuove un worker dal sistema
func (wm *WorkerManager) RemoveWorker(name string) error {
	wm.mutex.Lock()
	defer wm.mutex.Unlock()

	config, exists := wm.workers[name]
	if !exists {
		return fmt.Errorf("worker %s non trovato", name)
	}

	// Ferma il worker
	config.Worker.Stop()

	// Rimuovi dal map
	delete(wm.workers, name)

	log.Printf("🗑️  Worker %s rimosso", name)
	return nil
}

// Start avvia il WorkerManager e tutti i worker registrati
func (wm *WorkerManager) Start() {
	wm.mutex.Lock()
	if wm.isRunning {
		wm.mutex.Unlock()
		log.Println("⚠️  WorkerManager già in esecuzione")
		return
	}
	wm.isRunning = true
	wm.mutex.Unlock()

	log.Println("🚀 Avvio WorkerManager...")

	// Mostra worker registrati
	wm.mutex.RLock()
	enabledCount := 0
	for name, config := range wm.workers {
		if config.Enabled {
			enabledCount++
			log.Printf("   ✅ %s: %s (Schedule: %s)", name, config.Description, config.Schedule)
		} else {
			log.Printf("   ⚠️  %s: %s (DISABILITATO)", name, config.Description)
		}
	}
	wm.mutex.RUnlock()

	if enabledCount == 0 {
		log.Println("⚠️  Nessun worker abilitato trovato!")
		return
	}

	// Avvia il cron
	wm.cron.Start()
	log.Printf("✅ WorkerManager avviato con %d worker attivi", enabledCount)

	// Setup graceful shutdown
	wm.setupGracefulShutdown()
}

// Stop ferma tutti i worker e il cron
func (wm *WorkerManager) Stop() {
	wm.mutex.Lock()
	defer wm.mutex.Unlock()

	if !wm.isRunning {
		return
	}

	log.Println("🛑 Arresto WorkerManager...")

	// Ferma il cron
	ctx := wm.cron.Stop()
	select {
	case <-ctx.Done():
		log.Println("✅ Cron fermato correttamente")
	case <-time.After(30 * time.Second):
		log.Println("⚠️  Timeout arresto cron")
	}

	// Ferma tutti i worker
	for name, config := range wm.workers {
		log.Printf("🛑 Fermando worker %s...", name)
		config.Worker.Stop()
	}

	// Cancella il context
	wm.cancel()
	wm.isRunning = false

	log.Println("✅ WorkerManager fermato")
}

// GetWorkerStatus restituisce lo stato di tutti i worker
func (wm *WorkerManager) GetWorkerStatus() map[string]bool {
	wm.mutex.RLock()
	defer wm.mutex.RUnlock()

	status := make(map[string]bool)
	for name, config := range wm.workers {
		status[name] = config.Enabled
	}
	return status
}

// setupGracefulShutdown configura la gestione dei segnali per spegnimento pulito
func (wm *WorkerManager) setupGracefulShutdown() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		log.Println("🛑 Segnale di arresto ricevuto, spegnimento pulito...")
		wm.Stop()
		os.Exit(0)
	}()
}

// ===================================================================
// CONFIGURAZIONE WORKER - AGGIUNGI QUI I TUOI WORKER
// ===================================================================

// InitializeWorkers configura e avvia tutti i worker del sistema
func InitializeWorkers() *WorkerManager {
	log.Println("🔧 Inizializzazione sistema worker...")

	// Crea il WorkerManager
	manager := NewWorkerManager()

	// ====================================================================
	// 🔥 TRADING WORKERS
	// ====================================================================

	// Worker principale per il trading system DOGE
	dogeWorker := NewDogeTradingSystemWorker()
	dogeConfig := &WorkerConfig{
		Name:        "doge-trading-system",
		Schedule:    "0 0 * * * *", // Ogni ora al secondo 0
		Worker:      dogeWorker,
		Enabled:     true, // ✅ ABILITATO - Cambia a false per disabilitare
		Description: "Sistema di trading automatico per DOGEUSDT",
	}

	if err := manager.RegisterWorker(dogeConfig); err != nil {
		log.Printf("❌ Errore registrazione DOGE worker: %v", err)
	}
	// CRON EXPRESSIONS UTILI:
	// - "0 * * * * *"     = Ogni minuto
	// - "0 */5 * * * *"   = Ogni 5 minuti
	// - "0 0 * * * *"     = Ogni ora
	// - "0 0 9 * * *"     = Ogni giorno alle 9:00
	// - "0 0 9 * * 1"     = Ogni lunedì alle 9:00
	// - "0 0 9 1 * *"     = Il primo di ogni mese alle 9:00

	log.Printf("✅ Sistema worker inizializzato con %d worker registrati", len(manager.workers))
	return manager
}

// StartWorkerSystem è la funzione principale per avviare tutto il sistema worker
func StartWorkerSystem() {
	log.Println("🎯 === AVVIO SISTEMA WORKER TRADING ===")

	// Inizializza e avvia il sistema
	manager := InitializeWorkers()
	manager.Start()

	// Il sistema rimarrà in esecuzione fino a ricevere un segnale di stop
	// o fino a quando non viene chiamato manager.Stop()
	log.Println("✅ Sistema worker avviato. Premi Ctrl+C per fermare.")

	// Mantieni il programma in esecuzione
	select {}
}
