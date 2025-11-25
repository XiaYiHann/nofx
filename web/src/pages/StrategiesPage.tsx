import { useState, useEffect } from 'react'
import { useLanguage } from '../contexts/LanguageContext'
import { t } from '../i18n/translations'
import {
  Plus,
  Edit,
  Trash2,
  FileText,
  Lock,
  User,
  BookOpen,
} from 'lucide-react'
import StrategyEditor from '../components/StrategyEditor'
import { getAuthHeaders } from '../lib/api'

interface Strategy {
  name: string
  content?: string
  isSystem: boolean
  fileName?: string
}

export default function StrategiesPage() {
  const { language } = useLanguage()
  const [strategies, setStrategies] = useState<Strategy[]>([])
  const [loading, setLoading] = useState(true)
  const [isEditorOpen, setIsEditorOpen] = useState(false)
  const [editingStrategy, setEditingStrategy] = useState<Strategy | null>(null)
  const [viewingStrategy, setViewingStrategy] = useState<Strategy | null>(null)

  useEffect(() => {
    loadStrategies()
  }, [])

  const requireAuthHeaders = () => {
    const token = localStorage.getItem('auth_token')
    if (!token) {
      alert(
        language === 'zh'
          ? '登录已过期，请重新登录后再试'
          : 'Login expired, please sign in again and retry.'
      )
      return null
    }
    return getAuthHeaders()
  }

  const loadStrategies = async () => {
    try {
      const response = await fetch('/api/prompt-templates')

      if (response.ok) {
        const data = await response.json()
        const allStrategies: Strategy[] = data.templates.map((t: any) => ({
          name: t.name,
          isSystem: !t.name.startsWith('user_'),
          fileName: t.name,
        }))
        setStrategies(allStrategies)
      }
    } catch (error) {
      console.error('Failed to load strategies:', error)
    } finally {
      setLoading(false)
    }
  }

  const handleCreateNew = () => {
    setEditingStrategy(null)
    setIsEditorOpen(true)
  }

  const handleEdit = async (strategy: Strategy) => {
    try {
      const response = await fetch(`/api/prompt-templates/${strategy.name}`)

      if (response.ok) {
        const data = await response.json()
        setEditingStrategy({
          ...strategy,
          content: data.content,
        })
        setIsEditorOpen(true)
      }
    } catch (error) {
      console.error('Failed to load strategy content:', error)
    }
  }

  const handleView = async (strategy: Strategy) => {
    try {
      const response = await fetch(`/api/prompt-templates/${strategy.name}`)

      if (response.ok) {
        const data = await response.json()
        setViewingStrategy({
          ...strategy,
          content: data.content,
        })
      }
    } catch (error) {
      console.error('Failed to load strategy content:', error)
    }
  }

  const handleDelete = async (strategy: Strategy) => {
    if (!confirm(t('confirmDelete', language))) {
      return
    }

    try {
      const headers = requireAuthHeaders()
      if (!headers) return
      // Extract pure strategy name from template ID (removes user_<userid>_ prefix and .txt suffix)
      // e.g., "user_123_mystrategy" -> "mystrategy"
      const parts = strategy.name.split('_')
      let strategyName = strategy.name

      // If it starts with user_, remove the first two parts (user and userid)
      if (parts.length >= 3 && parts[0] === 'user') {
        strategyName = parts.slice(2).join('_')
      }

      // Remove .txt suffix if present
      strategyName = strategyName.replace(/\.txt$/, '')

      const response = await fetch(`/api/strategies/${strategyName}`, {
        method: 'DELETE',
        headers,
      })

      if (response.ok) {
        alert(t('strategyDeletedSuccess', language))
        loadStrategies()
      } else {
        const error = await response.json()
        alert(error.error || 'Failed to delete strategy')
      }
    } catch (error) {
      console.error('Failed to delete strategy:', error)
      alert('Failed to delete strategy')
    }
  }

  const handleSave = async (name: string, content: string) => {
    try {
      const isUpdate = editingStrategy !== null

      const url = isUpdate ? `/api/strategies/${name}` : '/api/strategies'
      const method = isUpdate ? 'PUT' : 'POST'

      const headers = requireAuthHeaders()
      if (!headers) return

      headers['Content-Type'] = 'application/json'

      const response = await fetch(url, {
        method,
        headers,
        body: JSON.stringify({
          name,
          content,
          description: '',
        }),
      })

      if (response.ok) {
        alert(
          t(
            isUpdate ? 'strategyUpdatedSuccess' : 'strategyCreatedSuccess',
            language
          )
        )
        setIsEditorOpen(false)
        loadStrategies()
      } else {
        const error = await response.json()
        alert(error.error || 'Failed to save strategy')
      }
    } catch (error) {
      console.error('Failed to save strategy:', error)
      alert('Failed to save strategy')
    }
  }

  const systemStrategies = strategies.filter((s) => s.isSystem)
  const userStrategies = strategies.filter((s) => !s.isSystem)

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-screen bg-[var(--background)]">
        <div className="text-[var(--text-secondary)]">Loading...</div>
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-[var(--background)] text-[var(--text-primary)] px-4 py-10">
      <div className="max-w-6xl mx-auto space-y-8">
        {/* Header */}
        <div className="space-y-3">
          <div className="flex items-center gap-3">
            <BookOpen className="w-8 h-8 text-[var(--binance-yellow)]" />
            <h1 className="text-3xl font-bold">
              {t('strategyManagement', language)}
            </h1>
          </div>
          <p className="text-[var(--text-secondary)] max-w-3xl">
            {language === 'zh'
              ? '创建和管理您的交易策略，策略将自动保存到 prompts 文件夹'
              : 'Create and manage your trading strategies, automatically saved to prompts folder'}
          </p>
        </div>

        {/* Create Button */}
        <div>
          <button
            onClick={handleCreateNew}
            className="btn-binance inline-flex items-center gap-2 text-base"
          >
            <Plus className="w-5 h-5" />
            {t('createStrategy', language)}
          </button>
        </div>

        {/* User Strategies */}
        <section>
          <div className="flex items-center gap-2 mb-4">
            <User className="w-5 h-5 text-[var(--binance-green)]" />
            <h2 className="text-xl font-semibold tracking-wide">
              {t('myStrategies', language)}
            </h2>
          </div>

          {userStrategies.length === 0 ? (
            <div className="binance-card p-12 text-center border border-dashed border-[var(--panel-border)]">
              <FileText className="w-16 h-16 text-[var(--text-secondary)] mx-auto mb-4" />
              <h3 className="text-lg font-medium mb-2">
                {t('noStrategies', language)}
              </h3>
              <p className="text-[var(--text-secondary)]">
                {t('noStrategiesDesc', language)}
              </p>
            </div>
          ) : (
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
              {userStrategies.map((strategy) => (
                <StrategyCard
                  key={strategy.name}
                  strategy={strategy}
                  language={language}
                  onEdit={handleEdit}
                  onDelete={handleDelete}
                  onView={handleView}
                />
              ))}
            </div>
          )}
        </section>

        {/* System Templates */}
        <section>
          <div className="flex items-center gap-2 mb-4">
            <Lock className="w-5 h-5 text-[var(--binance-yellow)]" />
            <h2 className="text-xl font-semibold tracking-wide">
              {t('systemTemplates', language)}
            </h2>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {systemStrategies.map((strategy) => (
              <StrategyCard
                key={strategy.name}
                strategy={strategy}
                language={language}
                onView={handleView}
              />
            ))}
          </div>
        </section>
      </div>

      {/* Strategy Editor Modal */}
      {isEditorOpen && (
        <StrategyEditor
          strategy={editingStrategy}
          onSave={handleSave}
          onClose={() => setIsEditorOpen(false)}
          language={language}
        />
      )}

      {/* View Modal */}
      {viewingStrategy && (
        <div className="fixed inset-0 bg-black/80 flex items-center justify-center z-50 p-4">
          <div className="bg-[var(--panel-bg)] border border-[var(--panel-border)] rounded-xl max-w-4xl w-full max-h-[90vh] overflow-hidden flex flex-col shadow-xl">
            <div className="p-6 border-b border-[var(--panel-border)] flex items-center justify-between">
              <h3 className="text-xl font-semibold flex items-center gap-2">
                <FileText className="w-5 h-5 text-[var(--binance-yellow)]" />
                {viewingStrategy.name}
                {viewingStrategy.isSystem && (
                  <span className="badge badge-yellow text-xs">
                    {t('systemTemplate', language)}
                  </span>
                )}
              </h3>
              <button
                onClick={() => setViewingStrategy(null)}
                className="text-[var(--text-secondary)] hover:text-[var(--text-primary)]"
                aria-label="Close"
              >
                ✕
              </button>
            </div>
            <div className="p-6 overflow-y-auto flex-1 bg-[var(--background)]">
              <pre className="text-sm text-[var(--text-primary)] whitespace-pre-wrap font-mono bg-[var(--background)] border border-[var(--panel-border)] rounded-lg p-4">
                {viewingStrategy.content}
              </pre>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

interface StrategyCardProps {
  strategy: Strategy
  language: 'en' | 'zh'
  onEdit?: (strategy: Strategy) => void
  onDelete?: (strategy: Strategy) => void
  onView: (strategy: Strategy) => void
}

function StrategyCard({
  strategy,
  language,
  onEdit,
  onDelete,
  onView,
}: StrategyCardProps) {
  const isSystem = strategy.isSystem
  const displayName = strategy.name.replace(/^user_[^_]+_/, '')

  return (
    <div className="binance-card p-5 flex flex-col h-full">
      <div className="flex items-start justify-between mb-3">
        <div className="flex-1 space-y-2">
          <h3 className="text-lg font-semibold flex items-center gap-2">
            {displayName}
            {isSystem && (
              <Lock className="w-4 h-4 text-[var(--binance-yellow)]" />
            )}
          </h3>
          <span
            className={`badge text-xs ${
              isSystem
                ? 'badge-yellow'
                : 'bg-[var(--binance-green-bg)] border-[var(--binance-green-border)] text-[var(--binance-green)]'
            }`}
          >
            {t(isSystem ? 'systemTemplate' : 'userStrategy', language)}
          </span>
        </div>
      </div>

      <div className="mt-auto flex gap-2">
        <button
          onClick={() => onView(strategy)}
          className="flex-1 px-3 py-2 border border-[var(--panel-border)] rounded text-sm text-[var(--text-primary)] bg-[var(--panel-bg)] hover:border-[var(--binance-yellow)] hover:text-[var(--binance-yellow)] transition-colors flex items-center justify-center gap-1"
        >
          <FileText className="w-4 h-4" />
          {t('viewTemplate', language)}
        </button>

        {!isSystem && onEdit && (
          <button
            onClick={() => onEdit(strategy)}
            className="px-3 py-2 border border-[var(--binance-yellow)] text-[var(--binance-yellow)] text-sm rounded transition-colors hover:bg-[var(--binance-yellow-glow)]"
            aria-label="Edit strategy"
          >
            <Edit className="w-4 h-4" />
          </button>
        )}

        {!isSystem && onDelete && (
          <button
            onClick={() => onDelete(strategy)}
            className="px-3 py-2 border border-[var(--binance-red-border)] text-[var(--binance-red)] text-sm rounded transition-colors bg-[var(--binance-red-bg)] hover:border-[var(--binance-red)]"
            aria-label="Delete strategy"
          >
            <Trash2 className="w-4 h-4" />
          </button>
        )}
      </div>
    </div>
  )
}
