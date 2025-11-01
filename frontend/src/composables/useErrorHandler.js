// frontend/src/composables/useErrorHandler.js
/**
 * @module useErrorHandler
 * @description Composable centralizzato per la gestione degli errori nell'applicazione.
 * Fornisce funzionalità per logging, notifiche utente e tracciamento degli errori.
 */
import { ref } from 'vue';
import { useNotifications } from './useNotifications';

const lastError = ref(null);
const errorHistory = ref([]);
const MAX_ERROR_HISTORY = 50;

/**
 * Tipi di errore supportati con configurazione predefinita
 */
const ERROR_TYPES = {
  NETWORK: {
    icon: 'wifi_off',
    defaultMessage: 'Errore di rete. Verifica la connessione.'
  },
  VALIDATION: {
    icon: 'warning',
    defaultMessage: 'I dati forniti non sono validi.'
  },
  PERMISSION: {
    icon: 'block',
    defaultMessage: 'Non hai i permessi necessari per questa operazione.'
  },
  NOT_FOUND: {
    icon: 'search_off',
    defaultMessage: 'Risorsa non trovata.'
  },
  TIMEOUT: {
    icon: 'schedule',
    defaultMessage: 'Operazione scaduta. Riprova.'
  },
  SERVER: {
    icon: 'dns',
    defaultMessage: 'Errore del server. Riprova più tardi.'
  },
  GENERIC: {
    icon: 'error',
    defaultMessage: 'Si è verificato un errore imprevisto.'
  }
};

/**
 * Composable per la gestione centralizzata degli errori.
 *
 * @returns {{
 *   handleError: Function,
 *   handleWarning: Function,
 *   clearLastError: Function,
 *   lastError: import('vue').Ref<Object|null>,
 *   errorHistory: import('vue').Ref<Array<Object>>,
 *   inferErrorType: Function
 * }}
 *
 * @example
 * const { handleError } = useErrorHandler();
 * try {
 *   await someOperation();
 * } catch (error) {
 *   handleError(error, 'Impossibile completare l\'operazione');
 * }
 */
export function useErrorHandler() {
  const { addNotification } = useNotifications();

  /**
   * Inferisce il tipo di errore dall'oggetto error o dal messaggio.
   *
   * @param {Error|string} error - L'errore da analizzare.
   * @returns {string} Il tipo di errore identificato.
   */
  const inferErrorType = (error) => {
    const errorMessage = error instanceof Error ? error.message : String(error);
    const lowerMessage = errorMessage.toLowerCase();

    if (lowerMessage.includes('network') || lowerMessage.includes('fetch') || lowerMessage.includes('connection')) {
      return 'NETWORK';
    }
    if (lowerMessage.includes('timeout') || lowerMessage.includes('timed out')) {
      return 'TIMEOUT';
    }
    if (lowerMessage.includes('not found') || lowerMessage.includes('404')) {
      return 'NOT_FOUND';
    }
    if (lowerMessage.includes('permission') || lowerMessage.includes('unauthorized') || lowerMessage.includes('forbidden')) {
      return 'PERMISSION';
    }
    if (lowerMessage.includes('validation') || lowerMessage.includes('invalid')) {
      return 'VALIDATION';
    }
    if (lowerMessage.includes('server') || lowerMessage.includes('500') || lowerMessage.includes('503')) {
      return 'SERVER';
    }

    return 'GENERIC';
  };

  /**
   * Formatta un messaggio di errore per l'utente finale.
   * Rimuove dettagli tecnici e fornisce informazioni chiare.
   *
   * @param {string} context - Il contesto dell'errore.
   * @param {string} errorType - Il tipo di errore identificato.
   * @returns {string} Il messaggio formattato.
   */
  const formatUserMessage = (context, errorType) => {
    const errorConfig = ERROR_TYPES[errorType] || ERROR_TYPES.GENERIC;

    // Se il context è specifico, usalo; altrimenti usa il messaggio di default
    if (context && context !== 'An unexpected error occurred') {
      return context;
    }

    return errorConfig.defaultMessage;
  };

  /**
   * Gestisce un errore loggandolo e mostrando una notifica user-friendly.
   *
   * @param {Error|string} error - L'errore da gestire.
   * @param {string} [context='An unexpected error occurred'] - Messaggio contestuale per l'utente.
   * @param {Object} [options={}] - Opzioni aggiuntive per la gestione dell'errore.
   * @param {boolean} [options.showNotification=true] - Se mostrare la notifica all'utente.
   * @param {boolean} [options.logToConsole=true] - Se loggare l'errore in console.
   * @param {string} [options.errorType] - Tipo di errore forzato (se non inferito).
   */
  const handleError = (error, context = 'An unexpected error occurred', options = {}) => {
    const {
      showNotification = true,
      logToConsole = true,
      errorType: forcedErrorType = null
    } = options;

    const errorMessage = error instanceof Error ? error.message : String(error);
    const errorType = forcedErrorType || inferErrorType(error);
    const userMessage = formatUserMessage(context, errorType);
    const fullMessage = `${context}: ${errorMessage}`;

    // Log completo per debugging
    if (logToConsole) {
      console.error(`[ErrorHandler] ${fullMessage}`, {
        type: errorType,
        context,
        originalError: error,
        timestamp: new Date().toISOString()
      });
    }

    // Memorizza l'ultimo errore
    const errorRecord = {
      message: fullMessage,
      userMessage,
      type: errorType,
      error,
      context,
      timestamp: Date.now()
    };

    lastError.value = errorRecord;

    // Aggiungi alla storia degli errori
    errorHistory.value.unshift(errorRecord);
    if (errorHistory.value.length > MAX_ERROR_HISTORY) {
      errorHistory.value = errorHistory.value.slice(0, MAX_ERROR_HISTORY);
    }

    // Mostra notifica all'utente
    if (showNotification) {
      addNotification({
        message: userMessage,
        type: 'error',
      });
    }
  };

  /**
   * Gestisce un warning (avviso) invece di un errore vero e proprio.
   *
   * @param {string} message - Il messaggio di warning.
   * @param {Object} [options={}] - Opzioni aggiuntive.
   * @param {boolean} [options.showNotification=true] - Se mostrare la notifica.
   * @param {boolean} [options.logToConsole=true] - Se loggare in console.
   */
  const handleWarning = (message, options = {}) => {
    const {
      showNotification = true,
      logToConsole = true
    } = options;

    if (logToConsole) {
      console.warn(`[ErrorHandler] ${message}`, {
        timestamp: new Date().toISOString()
      });
    }

    if (showNotification) {
      addNotification({
        message,
        type: 'warning',
      });
    }
  };

  /**
   * Pulisce l'ultimo errore memorizzato.
   */
  const clearLastError = () => {
    lastError.value = null;
  };

  /**
   * Wrapper per gestire errori async in modo sicuro.
   *
   * @param {Function} asyncFn - Funzione asincrona da eseguire.
   * @param {string} errorContext - Contesto dell'errore per messaggi user-friendly.
   * @returns {Promise<any>} Il risultato della funzione o undefined in caso di errore.
   *
   * @example
   * const result = await withErrorHandling(
   *   () => fetchData(),
   *   'Impossibile caricare i dati'
   * );
   */
  const withErrorHandling = async (asyncFn, errorContext) => {
    try {
      return await asyncFn();
    } catch (error) {
      handleError(error, errorContext);
      return undefined;
    }
  };

  return {
    handleError,
    handleWarning,
    clearLastError,
    withErrorHandling,
    lastError,
    errorHistory,
    inferErrorType
  };
}
