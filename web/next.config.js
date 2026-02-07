/** @type {import('next').NextConfig} */

const isStaticExport = process.env.STATIC_EXPORT === 'true'

const nextConfig = {
  // 静态导出模式：生成纯静态 HTML/CSS/JS，嵌入 Go 二进制
  ...(isStaticExport ? { output: 'export' } : {}),

  // 开发模式：反向代理到 Go 后端（静态导出不支持 rewrites）
  ...(!isStaticExport ? {
    async rewrites() {
      return [
        {
          source: '/api/:path*',
          destination: 'http://localhost:8080/api/:path*',
        },
        {
          source: '/ws/:path*',
          destination: 'http://localhost:8080/ws/:path*',
        },
        {
          source: '/ttyd/:path*',
          destination: 'http://localhost:8080/ttyd/:path*',
        },
      ]
    },
  } : {}),
}

module.exports = nextConfig
