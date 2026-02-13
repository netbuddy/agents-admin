/** @type {import('next').NextConfig} */

const isStaticExport = process.env.STATIC_EXPORT === 'true'

const nextConfig = {
  // 静态导出模式：生成纯静态 HTML/CSS/JS，嵌入 Go 二进制
  ...(isStaticExport ? { output: 'export' } : {}),

  // 开发模式 rewrites（仅当直接访问 Next.js :3002 时使用）
  // 推荐通过 Go 服务器 :8080 统一访问（Go 自动代理前端到 Next.js）
  ...(!isStaticExport ? {
    async rewrites() {
      const apiTarget = process.env.API_SERVER_URL || 'http://localhost:8080'
      return [
        {
          source: '/api/:path*',
          destination: `${apiTarget}/api/:path*`,
        },
        {
          source: '/ws/:path*',
          destination: `${apiTarget}/ws/:path*`,
        },
        {
          source: '/ttyd/:path*',
          destination: `${apiTarget}/ttyd/:path*`,
        },
        {
          source: '/terminal/:path*',
          destination: `${apiTarget}/terminal/:path*`,
        },
      ]
    },
  } : {}),
}

module.exports = nextConfig
