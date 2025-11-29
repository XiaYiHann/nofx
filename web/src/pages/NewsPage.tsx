import { useState } from 'react'
import useSWR from 'swr'
import { httpClient } from '../lib/httpClient'
import { NewsItem } from '../types'
import { Newspaper, ExternalLink, Clock, Flame } from 'lucide-react'
import { formatDistanceToNow } from 'date-fns'

const fetcher = (url: string) =>
  httpClient.get<{ news: NewsItem[] }>(url).then((res) => res.data?.news || [])

export default function NewsPage() {
  const [activeTab, setActiveTab] = useState<'all' | 'crypto' | 'tech'>('all')
  const {
    data: news,
    error,
    isLoading,
  } = useSWR<NewsItem[]>(`/api/news?category=${activeTab}`, fetcher)

  const tabs = [
    { id: 'all', label: 'All News' },
    { id: 'crypto', label: 'Crypto' },
    { id: 'tech', label: 'Tech' },
  ]

  return (
    <div className="container mx-auto p-6 max-w-7xl">
      <div className="flex flex-col space-y-6">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div className="flex items-center space-x-3">
            <div className="p-3 bg-blue-500/10 rounded-xl border border-blue-500/20">
              <Newspaper className="w-8 h-8 text-blue-400" />
            </div>
            <div>
              <h1 className="text-2xl font-bold text-white">News Center</h1>
              <p className="text-gray-400 text-sm">
                Real-time updates from CryptoPanic & Hacker News
              </p>
            </div>
          </div>
        </div>

        {/* Tabs */}
        <div className="flex space-x-1 bg-white/5 p-1 rounded-lg w-fit border border-white/10">
          {tabs.map((tab) => (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id as any)}
              className={`px-4 py-2 rounded-md text-sm font-medium transition-all ${
                activeTab === tab.id
                  ? 'bg-blue-500 text-white shadow-lg shadow-blue-500/20'
                  : 'text-gray-400 hover:text-white hover:bg-white/5'
              }`}
            >
              {tab.label}
            </button>
          ))}
        </div>

        {/* Content */}
        {isLoading ? (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            {[...Array(6)].map((_, i) => (
              <div
                key={i}
                className="h-48 bg-white/5 rounded-xl animate-pulse border border-white/10"
              />
            ))}
          </div>
        ) : error ? (
          <div className="text-center py-20 bg-red-500/5 rounded-xl border border-red-500/10">
            <p className="text-red-400">
              Failed to load news. Please try again later.
            </p>
          </div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            {news?.map((item) => (
              <a
                key={item.id}
                href={item.url}
                target="_blank"
                rel="noopener noreferrer"
                className="group relative flex flex-col p-5 bg-white/5 hover:bg-white/10 border border-white/10 hover:border-blue-500/30 rounded-xl transition-all duration-300 hover:-translate-y-1 hover:shadow-xl hover:shadow-blue-500/10"
              >
                <div className="flex justify-between items-start mb-3">
                  <span
                    className={`px-2 py-1 text-xs font-medium rounded-full border ${
                      item.source === 'CryptoPanic'
                        ? 'bg-purple-500/10 text-purple-400 border-purple-500/20'
                        : 'bg-orange-500/10 text-orange-400 border-orange-500/20'
                    }`}
                  >
                    {item.source}
                  </span>
                  {item.score > 0 && (
                    <div className="flex items-center text-xs text-yellow-500 font-medium">
                      <Flame className="w-3 h-3 mr-1" />
                      {item.score}
                    </div>
                  )}
                </div>

                <h3 className="text-lg font-semibold text-gray-100 mb-2 line-clamp-2 group-hover:text-blue-400 transition-colors">
                  {item.title}
                </h3>

                <p className="text-gray-400 text-sm mb-4 line-clamp-3 flex-grow">
                  {item.summary}
                </p>

                <div className="flex items-center justify-between text-xs text-gray-500 mt-auto pt-4 border-t border-white/5">
                  <div className="flex items-center">
                    <Clock className="w-3 h-3 mr-1" />
                    {formatDistanceToNow(new Date(item.published_at), {
                      addSuffix: true,
                    })}
                  </div>
                  <ExternalLink className="w-4 h-4 opacity-0 group-hover:opacity-100 transition-opacity text-blue-400" />
                </div>
              </a>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
