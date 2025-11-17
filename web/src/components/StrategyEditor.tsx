import { useState, useEffect } from 'react'
import { t } from '../i18n/translations'
import { X, Save, FileText, Eye, BookOpen } from 'lucide-react'
import { getAuthHeaders } from '../lib/api'

interface StrategyEditorProps {
  strategy: {
    name: string
    content?: string
    isSystem: boolean
  } | null
  onSave: (name: string, content: string) => void
  onClose: () => void
  language: 'en' | 'zh'
}

export default function StrategyEditor({
  strategy,
  onSave,
  onClose,
  language,
}: StrategyEditorProps) {
  const [name, setName] = useState('')
  const [content, setContent] = useState('')
  const [showPreview, setShowPreview] = useState(false)
  const [templates, setTemplates] = useState<any[]>([])
  const [showTemplateSelector, setShowTemplateSelector] = useState(false)

  const isEditing = strategy !== null

  const loadTemplates = async () => {
    try {
      const response = await fetch('/api/prompt-templates', {
        headers: getAuthHeaders(),
      })

      if (response.ok) {
        const data = await response.json()
        setTemplates(
          data.templates.filter((t: any) => !t.name.startsWith('user_'))
        )
      }
    } catch (error) {
      console.error('Failed to load templates:', error)
    }
  }

  const loadTemplateContent = async (templateName: string) => {
    try {
      const response = await fetch(`/api/prompt-templates/${templateName}`, {
        headers: getAuthHeaders(),
      })

      if (response.ok) {
        const data = await response.json()
        setContent(data.content)
        setShowTemplateSelector(false)
      }
    } catch (error) {
      console.error('Failed to load template content:', error)
    }
  }

  useEffect(() => {
    if (strategy) {
      setName(strategy.name.replace(/^user_[^_]+_/, ''))
      setContent(strategy.content || '')
    }

    // Load system templates for reference
    loadTemplates()
  }, [strategy])

  const handleSubmit = () => {
    if (!name.trim()) {
      alert(t('strategyNameRequired', language))
      return
    }

    if (!content.trim()) {
      alert(t('strategyContentRequired', language))
      return
    }

    // Validate name format
    if (!/^[\w\u4e00-\u9fa5]+$/.test(name)) {
      alert(t('strategyNameInvalid', language))
      return
    }

    onSave(name, content)
  }

  return (
    <div className="fixed inset-0 bg-black/80 flex items-center justify-center z-50 p-4">
      <div className="bg-[var(--panel-bg)] border border-[var(--panel-border)] rounded-xl max-w-6xl w-full max-h-[90vh] overflow-hidden flex flex-col shadow-xl">
        {/* Header */}
        <div className="p-6 border-b border-[var(--panel-border)] flex items-center justify-between">
          <h3 className="text-xl font-semibold flex items-center gap-2">
            <FileText className="w-5 h-5 text-[var(--binance-yellow)]" />
            {t(isEditing ? 'editStrategy' : 'createStrategy', language)}
          </h3>
          <button
            onClick={onClose}
            className="text-[var(--text-secondary)] hover:text-[var(--text-primary)] transition-colors"
            aria-label="Close"
          >
            <X className="w-6 h-6" />
          </button>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto p-6">
          <div className="space-y-6">
            {/* Strategy Name */}
            <div>
              <label className="block text-sm font-medium text-[var(--text-secondary)] mb-2">
                {t('strategyName', language)} *
              </label>
              <input
                type="text"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder={t('strategyNamePlaceholder', language)}
                disabled={isEditing}
                className="w-full px-4 py-2 bg-[var(--background)] border border-[var(--panel-border)] rounded-lg text-[var(--text-primary)] placeholder-[var(--text-secondary)] focus:outline-none focus:ring-2 focus:ring-[var(--binance-yellow)] disabled:opacity-50 disabled:cursor-not-allowed"
              />
              <p className="text-xs text-[var(--text-secondary)] mt-1">
                {language === 'zh'
                  ? '只能包含字母、数字、下划线和中文，编辑时不可修改'
                  : 'Letters, numbers, underscores and Chinese only, cannot be changed when editing'}
              </p>
            </div>

            {/* Tabs */}
            <div className="flex gap-2 border-b border-[var(--panel-border)]">
              <button
                onClick={() => setShowPreview(false)}
                className={`px-4 py-2 font-medium transition-colors ${
                  !showPreview
                    ? 'text-[var(--binance-yellow)] border-b-2 border-[var(--binance-yellow)]'
                    : 'text-[var(--text-secondary)] hover:text-[var(--text-primary)]'
                }`}
              >
                {t('strategyEditor', language)}
              </button>
              <button
                onClick={() => setShowPreview(true)}
                className={`px-4 py-2 font-medium transition-colors flex items-center gap-1 ${
                  showPreview
                    ? 'text-[var(--binance-yellow)] border-b-2 border-[var(--binance-yellow)]'
                    : 'text-[var(--text-secondary)] hover:text-[var(--text-primary)]'
                }`}
              >
                <Eye className="w-4 h-4" />
                {t('previewStrategy', language)}
              </button>
              <button
                onClick={() => setShowTemplateSelector(!showTemplateSelector)}
                className="ml-auto px-4 py-2 font-medium text-[var(--text-secondary)] hover:text-[var(--text-primary)] transition-colors flex items-center gap-1"
              >
                <BookOpen className="w-4 h-4" />
                {t('referenceTemplates', language)}
              </button>
            </div>

            {/* Template Selector */}
            {showTemplateSelector && (
              <div className="bg-[var(--background)] border border-[var(--panel-border)] rounded-lg p-4">
                <h4 className="text-sm font-medium text-[var(--text-secondary)] mb-3">
                  {t('systemTemplates', language)}
                </h4>
                <div className="grid grid-cols-2 md:grid-cols-3 gap-2">
                  {templates.map((template) => (
                    <button
                      key={template.name}
                      onClick={() => loadTemplateContent(template.name)}
                      className="px-3 py-2 bg-[var(--background)] border border-[var(--panel-border)] hover:border-[var(--binance-yellow)] text-[var(--text-primary)] text-sm rounded transition-colors text-left"
                    >
                      {template.name}
                    </button>
                  ))}
                </div>
              </div>
            )}

            {/* Editor or Preview */}
            {!showPreview ? (
              <div>
                <label className="block text-sm font-medium text-[var(--text-secondary)] mb-2">
                  {t('strategyContent', language)} *
                </label>
                <textarea
                  value={content}
                  onChange={(e) => setContent(e.target.value)}
                  placeholder={t('strategyContentPlaceholder', language)}
                  className="w-full h-96 px-4 py-3 bg-[var(--background)] border border-[var(--panel-border)] rounded-lg text-[var(--text-primary)] placeholder-[var(--text-secondary)] focus:outline-none focus:ring-2 focus:ring-[var(--binance-yellow)] font-mono text-sm resize-none"
                />
                <p className="text-xs text-[var(--text-secondary)] mt-1">
                  {language === 'zh'
                    ? '提示：可以从系统模板加载后修改，或完全自定义'
                    : 'Tip: Load from system templates and modify, or create from scratch'}
                </p>
              </div>
            ) : (
              <div>
                <div className="bg-[var(--background)] border border-[var(--panel-border)] rounded-lg p-4 h-96 overflow-y-auto">
                  <pre className="text-sm text-[var(--text-primary)] whitespace-pre-wrap font-mono">
                    {content || (
                      <span className="text-[var(--text-secondary)]">
                        {language === 'zh'
                          ? '暂无内容预览'
                          : 'No content to preview'}
                      </span>
                    )}
                  </pre>
                </div>
              </div>
            )}
          </div>
        </div>

        {/* Footer */}
        <div className="p-6 border-t border-[var(--panel-border)] flex items-center justify-end gap-3">
          <button
            onClick={onClose}
            className="px-6 py-2 border border-[var(--panel-border)] rounded-lg text-[var(--text-primary)] hover:border-[var(--binance-yellow)] transition-colors"
          >
            {t('cancelEdit', language)}
          </button>
          <button
            onClick={handleSubmit}
            className="btn-binance inline-flex items-center gap-2"
          >
            <Save className="w-4 h-4" />
            {t('saveStrategy', language)}
          </button>
        </div>
      </div>
    </div>
  )
}
