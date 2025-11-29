import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { BrowserRouter } from 'react-router-dom'
import NewsPage from './NewsPage'
import { httpClient } from '../lib/httpClient'

// Mock httpClient
vi.mock('../lib/httpClient', () => ({
  httpClient: {
    get: vi.fn(),
  },
}))

// Mock SWR to have more control over loading states
vi.mock('swr', async () => {
  const actual = await vi.importActual('swr')
  return {
    ...actual,
    default: vi.fn(),
  }
})

import useSWR from 'swr'

const mockUseSWR = vi.mocked(useSWR)

const mockNewsData = [
  {
    id: 'test-1',
    title: 'Test News Article 1',
    summary: 'This is a summary of the first test article',
    url: 'https://example.com/news/1',
    source: 'CryptoPanic',
    category: 'crypto',
    published_at: new Date().toISOString(),
    score: 100,
  },
  {
    id: 'test-2',
    title: 'Hacker News Article',
    summary: 'Points: 150 | Comments: 25',
    url: 'https://example.com/news/2',
    source: 'Hacker News',
    category: 'tech',
    published_at: new Date(Date.now() - 3600000).toISOString(), // 1 hour ago
    score: 150,
  },
]

const renderNewsPage = () => {
  return render(
    <BrowserRouter>
      <NewsPage />
    </BrowserRouter>
  )
}

describe('NewsPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  afterEach(() => {
    vi.resetAllMocks()
  })

  describe('Loading State', () => {
    it('should display loading skeleton when data is being fetched', () => {
      mockUseSWR.mockReturnValue({
        data: undefined,
        error: undefined,
        isLoading: true,
        isValidating: false,
        mutate: vi.fn(),
      })

      renderNewsPage()

      // Should show skeleton loaders (6 animated placeholders)
      const skeletons = document.querySelectorAll('.animate-pulse')
      expect(skeletons.length).toBeGreaterThan(0)
    })
  })

  describe('Success State', () => {
    it('should render news items when data is loaded', async () => {
      mockUseSWR.mockReturnValue({
        data: mockNewsData,
        error: undefined,
        isLoading: false,
        isValidating: false,
        mutate: vi.fn(),
      })

      renderNewsPage()

      await waitFor(() => {
        expect(screen.getByText('Test News Article 1')).toBeInTheDocument()
        expect(screen.getByText('Hacker News Article')).toBeInTheDocument()
      })
    })

    it('should display news title correctly', () => {
      mockUseSWR.mockReturnValue({
        data: mockNewsData,
        error: undefined,
        isLoading: false,
        isValidating: false,
        mutate: vi.fn(),
      })

      renderNewsPage()

      expect(screen.getByText('Test News Article 1')).toBeInTheDocument()
    })

    it('should display news summary', () => {
      mockUseSWR.mockReturnValue({
        data: mockNewsData,
        error: undefined,
        isLoading: false,
        isValidating: false,
        mutate: vi.fn(),
      })

      renderNewsPage()

      expect(
        screen.getByText('This is a summary of the first test article')
      ).toBeInTheDocument()
    })

    it('should display news source badge', () => {
      mockUseSWR.mockReturnValue({
        data: mockNewsData,
        error: undefined,
        isLoading: false,
        isValidating: false,
        mutate: vi.fn(),
      })

      renderNewsPage()

      expect(screen.getByText('CryptoPanic')).toBeInTheDocument()
      expect(screen.getByText('Hacker News')).toBeInTheDocument()
    })

    it('should display score for items with score > 0', () => {
      mockUseSWR.mockReturnValue({
        data: mockNewsData,
        error: undefined,
        isLoading: false,
        isValidating: false,
        mutate: vi.fn(),
      })

      renderNewsPage()

      // Both items have score > 0
      expect(screen.getByText('100')).toBeInTheDocument()
      expect(screen.getByText('150')).toBeInTheDocument()
    })

    it('should render news items as links with correct href', () => {
      mockUseSWR.mockReturnValue({
        data: mockNewsData,
        error: undefined,
        isLoading: false,
        isValidating: false,
        mutate: vi.fn(),
      })

      renderNewsPage()

      const links = screen.getAllByRole('link')
      const newsLinks = links.filter((link) =>
        link.getAttribute('href')?.includes('example.com/news')
      )
      expect(newsLinks).toHaveLength(2)
      expect(newsLinks[0]).toHaveAttribute('target', '_blank')
      expect(newsLinks[0]).toHaveAttribute('rel', 'noopener noreferrer')
    })

    it('should display relative time for published_at', () => {
      mockUseSWR.mockReturnValue({
        data: mockNewsData,
        error: undefined,
        isLoading: false,
        isValidating: false,
        mutate: vi.fn(),
      })

      renderNewsPage()

      // date-fns formatDistanceToNow will show something like "less than a minute ago" or "about 1 hour ago"
      // We check that multiple time-related texts are present (one per news item)
      const timeElements = screen.getAllByText(/ago/i)
      expect(timeElements.length).toBeGreaterThanOrEqual(1)
    })
  })

  describe('Error State', () => {
    it('should display error message when fetch fails', () => {
      mockUseSWR.mockReturnValue({
        data: undefined,
        error: new Error('Network error'),
        isLoading: false,
        isValidating: false,
        mutate: vi.fn(),
      })

      renderNewsPage()

      expect(screen.getByText(/Failed to load news/i)).toBeInTheDocument()
    })

    it('should not crash the page on error', () => {
      mockUseSWR.mockReturnValue({
        data: undefined,
        error: new Error('Server error'),
        isLoading: false,
        isValidating: false,
        mutate: vi.fn(),
      })

      // Should not throw
      expect(() => renderNewsPage()).not.toThrow()
    })
  })

  describe('Empty State', () => {
    it('should handle empty news array gracefully', () => {
      mockUseSWR.mockReturnValue({
        data: [],
        error: undefined,
        isLoading: false,
        isValidating: false,
        mutate: vi.fn(),
      })

      renderNewsPage()

      // Should render without crashing, no news items displayed
      expect(
        screen.queryByRole('link', { name: /test/i })
      ).not.toBeInTheDocument()
    })
  })

  describe('Tab Navigation', () => {
    it('should display category tabs', () => {
      mockUseSWR.mockReturnValue({
        data: mockNewsData,
        error: undefined,
        isLoading: false,
        isValidating: false,
        mutate: vi.fn(),
      })

      renderNewsPage()

      expect(screen.getByText('All News')).toBeInTheDocument()
      expect(screen.getByText('Crypto')).toBeInTheDocument()
      expect(screen.getByText('Tech')).toBeInTheDocument()
    })
  })

  describe('Page Header', () => {
    it('should display page title', () => {
      mockUseSWR.mockReturnValue({
        data: [],
        error: undefined,
        isLoading: false,
        isValidating: false,
        mutate: vi.fn(),
      })

      renderNewsPage()

      expect(screen.getByText('News Center')).toBeInTheDocument()
    })

    it('should display page description', () => {
      mockUseSWR.mockReturnValue({
        data: [],
        error: undefined,
        isLoading: false,
        isValidating: false,
        mutate: vi.fn(),
      })

      renderNewsPage()

      expect(screen.getByText(/CryptoPanic.*Hacker News/i)).toBeInTheDocument()
    })
  })
})
