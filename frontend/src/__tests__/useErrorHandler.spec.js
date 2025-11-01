/**
 * @file useErrorHandler.spec.js
 * @description Test suite per il composable useErrorHandler che gestisce
 * la centralizzazione degli errori nell'applicazione.
 */
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { useErrorHandler } from '../composables/useErrorHandler'
import { useNotifications } from '../composables/useNotifications'

// Mock del composable useNotifications
vi.mock('../composables/useNotifications', () => ({
  useNotifications: vi.fn(() => ({
    addNotification: vi.fn()
  }))
}))

describe('useErrorHandler', () => {
  let errorHandler
  let mockAddNotification

  beforeEach(() => {
    // Reset dei mock prima di ogni test
    vi.clearAllMocks()

    // Configura il mock per useNotifications
    mockAddNotification = vi.fn()
    useNotifications.mockReturnValue({
      addNotification: mockAddNotification
    })

    // Crea una nuova istanza del composable
    errorHandler = useErrorHandler()

    // Reset dello stato globale
    errorHandler.lastError.value = null
    errorHandler.errorHistory.value = []

    // Mock di console.error e console.warn per evitare output nei test
    vi.spyOn(console, 'error').mockImplementation(() => {})
    vi.spyOn(console, 'warn').mockImplementation(() => {})
  })

  describe('handleError', () => {
    it('gestisce un errore generico con messaggio di default', () => {
      const error = new Error('Test error')
      errorHandler.handleError(error)

      expect(errorHandler.lastError.value).toBeTruthy()
      expect(errorHandler.lastError.value.type).toBe('GENERIC')
      expect(errorHandler.lastError.value.context).toBe('An unexpected error occurred')
      expect(mockAddNotification).toHaveBeenCalledWith({
        message: 'Si è verificato un errore imprevisto.',
        type: 'error'
      })
    })

    it('gestisce un errore con contesto personalizzato', () => {
      const error = new Error('Test error')
      const context = 'Impossibile caricare i dati'

      errorHandler.handleError(error, context)

      expect(errorHandler.lastError.value.context).toBe(context)
      expect(mockAddNotification).toHaveBeenCalledWith({
        message: context,
        type: 'error'
      })
    })

    it('inferisce correttamente il tipo di errore NETWORK', () => {
      const error = new Error('Network connection failed')
      errorHandler.handleError(error)

      expect(errorHandler.lastError.value.type).toBe('NETWORK')
      expect(mockAddNotification).toHaveBeenCalledWith({
        message: 'Errore di rete. Verifica la connessione.',
        type: 'error'
      })
    })

    it('inferisce correttamente il tipo di errore TIMEOUT', () => {
      const error = new Error('Request timed out')
      errorHandler.handleError(error)

      expect(errorHandler.lastError.value.type).toBe('TIMEOUT')
    })

    it('inferisce correttamente il tipo di errore NOT_FOUND', () => {
      const error = new Error('Resource not found')
      errorHandler.handleError(error)

      expect(errorHandler.lastError.value.type).toBe('NOT_FOUND')
    })

    it('inferisce correttamente il tipo di errore VALIDATION', () => {
      const error = new Error('Invalid input provided')
      errorHandler.handleError(error)

      expect(errorHandler.lastError.value.type).toBe('VALIDATION')
    })

    it('inferisce correttamente il tipo di errore PERMISSION', () => {
      const error = new Error('Unauthorized access')
      errorHandler.handleError(error)

      expect(errorHandler.lastError.value.type).toBe('PERMISSION')
    })

    it('gestisce errori come stringhe', () => {
      const errorString = 'Simple error message'
      errorHandler.handleError(errorString)

      expect(errorHandler.lastError.value).toBeTruthy()
      expect(errorHandler.lastError.value.message).toContain(errorString)
    })

    it('permette di forzare il tipo di errore tramite options', () => {
      const error = new Error('Test error')
      errorHandler.handleError(error, 'Custom context', { errorType: 'TIMEOUT' })

      expect(errorHandler.lastError.value.type).toBe('TIMEOUT')
    })

    it('non mostra notifica quando showNotification è false', () => {
      const error = new Error('Test error')
      errorHandler.handleError(error, 'Test', { showNotification: false })

      expect(mockAddNotification).not.toHaveBeenCalled()
    })

    it('non logga in console quando logToConsole è false', () => {
      const error = new Error('Test error')
      errorHandler.handleError(error, 'Test', { logToConsole: false })

      expect(console.error).not.toHaveBeenCalled()
    })

    it('aggiunge l\'errore alla storia degli errori', () => {
      const error1 = new Error('First error')
      const error2 = new Error('Second error')

      errorHandler.handleError(error1)
      errorHandler.handleError(error2)

      expect(errorHandler.errorHistory.value.length).toBe(2)
      expect(errorHandler.errorHistory.value[0].error.message).toBe('Second error')
      expect(errorHandler.errorHistory.value[1].error.message).toBe('First error')
    })

    it('limita la storia degli errori a MAX_ERROR_HISTORY', () => {
      // Crea 55 errori (supera il limite di 50)
      for (let i = 0; i < 55; i++) {
        errorHandler.handleError(new Error(`Error ${i}`))
      }

      expect(errorHandler.errorHistory.value.length).toBe(50)
      expect(errorHandler.errorHistory.value[0].error.message).toBe('Error 54')
    })
  })

  describe('handleWarning', () => {
    it('gestisce un warning con notifica', () => {
      const message = 'This is a warning'
      errorHandler.handleWarning(message)

      expect(mockAddNotification).toHaveBeenCalledWith({
        message,
        type: 'warning'
      })
      expect(console.warn).toHaveBeenCalled()
    })

    it('non mostra notifica quando showNotification è false', () => {
      errorHandler.handleWarning('Test warning', { showNotification: false })

      expect(mockAddNotification).not.toHaveBeenCalled()
    })

    it('non logga in console quando logToConsole è false', () => {
      errorHandler.handleWarning('Test warning', { logToConsole: false })

      expect(console.warn).not.toHaveBeenCalled()
    })
  })

  describe('clearLastError', () => {
    it('pulisce l\'ultimo errore memorizzato', () => {
      errorHandler.handleError(new Error('Test error'))
      expect(errorHandler.lastError.value).toBeTruthy()

      errorHandler.clearLastError()
      expect(errorHandler.lastError.value).toBeNull()
    })
  })

  describe('withErrorHandling', () => {
    it('restituisce il risultato della funzione asincrona in caso di successo', async () => {
      const asyncFn = vi.fn(async () => 'success result')
      const result = await errorHandler.withErrorHandling(asyncFn, 'Test context')

      expect(result).toBe('success result')
      expect(asyncFn).toHaveBeenCalled()
      expect(mockAddNotification).not.toHaveBeenCalled()
    })

    it('gestisce l\'errore e restituisce undefined in caso di fallimento', async () => {
      const error = new Error('Async error')
      const asyncFn = vi.fn(async () => {
        throw error
      })

      const result = await errorHandler.withErrorHandling(asyncFn, 'Test context')

      expect(result).toBeUndefined()
      expect(asyncFn).toHaveBeenCalled()
      expect(errorHandler.lastError.value).toBeTruthy()
      expect(mockAddNotification).toHaveBeenCalled()
    })
  })

  describe('inferErrorType', () => {
    it('identifica correttamente diversi tipi di errore', () => {
      const testCases = [
        { message: 'Network error', expected: 'NETWORK' },
        { message: 'Fetch failed', expected: 'NETWORK' },
        { message: 'Connection refused', expected: 'NETWORK' },
        { message: 'Request timeout', expected: 'TIMEOUT' },
        { message: 'Operation timed out', expected: 'TIMEOUT' },
        { message: 'Resource not found', expected: 'NOT_FOUND' },
        { message: 'Error 404', expected: 'NOT_FOUND' },
        { message: 'Unauthorized access', expected: 'PERMISSION' },
        { message: 'Permission denied', expected: 'PERMISSION' },
        { message: 'Forbidden', expected: 'PERMISSION' },
        { message: 'Validation failed', expected: 'VALIDATION' },
        { message: 'Invalid data', expected: 'VALIDATION' },
        { message: 'Server error 500', expected: 'SERVER' },
        { message: 'Service unavailable 503', expected: 'SERVER' },
        { message: 'Random error', expected: 'GENERIC' }
      ]

      testCases.forEach(({ message, expected }) => {
        const result = errorHandler.inferErrorType(new Error(message))
        expect(result).toBe(expected)
      })
    })

    it('gestisce errori come stringhe', () => {
      const result = errorHandler.inferErrorType('Network connection failed')
      expect(result).toBe('NETWORK')
    })
  })

  describe('integrazione completa', () => {
    it('gestisce un flusso completo di errori multipli', () => {
      // Simula errori di vario tipo
      errorHandler.handleError(new Error('Network error'), 'Loading data')
      errorHandler.handleWarning('Deprecated API used')
      errorHandler.handleError(new Error('Validation failed'), 'Saving form')

      // Verifica lo stato finale
      expect(errorHandler.errorHistory.value.length).toBe(2) // Solo gli errori, non i warning
      expect(errorHandler.lastError.value.type).toBe('VALIDATION')
      expect(mockAddNotification).toHaveBeenCalledTimes(3) // 2 errori + 1 warning
    })

    it('mantiene la storia degli errori anche dopo clearLastError', () => {
      errorHandler.handleError(new Error('Test error'))
      errorHandler.clearLastError()

      expect(errorHandler.lastError.value).toBeNull()
      expect(errorHandler.errorHistory.value.length).toBe(1)
    })
  })
})
