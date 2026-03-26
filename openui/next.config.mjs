/** @type {import('next').NextConfig} */
const nextConfig = {
  typescript: {
    ignoreBuildErrors: true,
  },
  images: {
    unoptimized: true,
  },
  // Allow self-signed certificates in development
  experimental: {
    serverActions: {
      allowedOrigins: ['localhost:3000'],
    },
  },
  output: 'standalone',
}

// Disable SSL verification for self-signed certificates in development
if (process.env.NODE_ENV === 'development' || process.env.ALLOW_SELF_SIGNED_CERTS === 'true') {
  process.env.NODE_TLS_REJECT_UNAUTHORIZED = '0';
}

export default nextConfig
