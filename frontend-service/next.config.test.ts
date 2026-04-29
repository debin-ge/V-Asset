import nextConfig from './next.config'

type RemotePattern = {
  protocol?: string
  hostname?: string
  pathname?: string
  search?: string
}

function wildcardToRegExp(pattern: string): RegExp {
  const escaped = pattern.replace(/[.+?^${}()|[\]\\]/g, '\\$&')
  const wildcardPattern = escaped
    .replace(/\*\*/g, '__DOUBLE_WILDCARD__')
    .replace(/\*/g, '[^.]*')
    .replace(/__DOUBLE_WILDCARD__/g, '.*')

  return new RegExp(`^${wildcardPattern}$`)
}

function isAllowedRemoteImage(src: string): boolean {
  const url = new URL(src)
  const remotePatterns = (nextConfig.images?.remotePatterns ?? []) as RemotePattern[]

  return remotePatterns.some((pattern) => {
    if (pattern.protocol && pattern.protocol !== url.protocol.replace(':', '')) {
      return false
    }

    if (pattern.hostname && !wildcardToRegExp(pattern.hostname).test(url.hostname)) {
      return false
    }

    if (pattern.search !== undefined && pattern.search !== url.search) {
      return false
    }

    return true
  })
}

describe('next image remote patterns', () => {
  it('keeps YouTube thumbnails allowed', () => {
    expect(
      isAllowedRemoteImage('https://i.ytimg.com/vi/dQw4w9WgXcQ/maxresdefault.jpg')
    ).toBe(true)
  })

  it('allows TikTok CDN thumbnails with signed query parameters', () => {
    const tiktokThumbnail =
      'https://p16-common-sign.tiktokcdn-us.com/tos-alisg-p-0037/osAbDIkT4IVBrj7QQEteCLWWeGDNerQ78nw2QA~tplv-tiktokx-origin.image?dr=9636&x-expires=1777618800&x-signature=MYzFEFtw374Skda%2BlBART4Kdico%3D&t=4d5b0474&ps=13740610&shp=81f88b70&shcp=43f4a2f9&idc=useast5'

    expect(isAllowedRemoteImage(tiktokThumbnail)).toBe(true)
  })

  it('allows Bilibili thumbnails served over http', () => {
    const bilibiliThumbnail =
      'http://i0.hdslb.com/bfs/archive/51f48444b14cfd42bed20f7d1828ce229dada35c.jpg'

    expect(isAllowedRemoteImage(bilibiliThumbnail)).toBe(true)
  })
})
