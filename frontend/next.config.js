/** @type {import('next').NextConfig} */
const nextConfig = {
  // Ensure proper TypeScript path resolution
  typescript: {
    // Ignore TypeScript errors during build for demo purposes
    ignoreBuildErrors: false,
  },
  // API proxy to Go backend (future enhancement)
  async rewrites() {
    return [
      {
        source: '/api/v1/:path*',
        destination: 'http://localhost:8080/api/v1/:path*',
      },
    ];
  },
};

module.exports = nextConfig;