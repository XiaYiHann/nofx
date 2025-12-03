export interface SystemConfig {
  beta_mode: boolean
  registration_enabled?: boolean
  dev_mode?: boolean // 测试模式标志（仅用于前端检测是否启用免登录调试）
}

let configPromise: Promise<SystemConfig> | null = null
let cachedConfig: SystemConfig | null = null

export function getSystemConfig(): Promise<SystemConfig> {
  if (cachedConfig) {
    return Promise.resolve(cachedConfig)
  }
  if (configPromise) {
    return configPromise
  }
  configPromise = fetch('/api/config')
    .then((res) => {
      if (!res.ok) {
        throw new Error(`HTTP ${res.status}`)
      }
      return res.json()
    })
    .then((data: SystemConfig) => {
      cachedConfig = data
      return data
    })
    .catch((error) => {
      console.warn('[config] Failed to fetch system config:', error.message)

      // 在开发环境中，如果后端不可用，自动启用 dev_mode
      // 这允许前端独立运行和测试，无需后端服务器
      const isDevelopment = import.meta.env.DEV
      if (isDevelopment) {
        console.log('[config] 🚧 Development mode: Backend unavailable, enabling dev_mode for frontend-only testing')
        const fallbackConfig: SystemConfig = {
          beta_mode: false,
          registration_enabled: true,
          dev_mode: true, // 自动启用开发模式
        }
        cachedConfig = fallbackConfig
        return fallbackConfig
      }

      // 生产环境中抛出错误
      throw error
    })
    .finally(() => {
      // Keep cachedConfig for reuse; allow re-fetch via explicit invalidation if added later
    })
  return configPromise
}
