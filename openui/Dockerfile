# Multi-stage Dockerfile for Next.js 16 - Optimized for linux/amd64
FROM node:20-slim AS base

# =============================================
# Stage 1: Dependencies
# =============================================
FROM base AS deps

# Install minimal system dependencies needed for native modules
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY package.json package-lock.json ./

# Install dependencies with frozen lockfile
RUN npm ci --include=dev --no-optional --frozen-lockfile

# =============================================
# Stage 2: Builder
# =============================================
FROM base AS builder

WORKDIR /app

COPY --from=deps /app/node_modules ./node_modules
COPY . .

ENV NEXT_TELEMETRY_DISABLED=1

# Uncomment the line below if Turbopack causes problems during build
# ENV NEXT_BUILD_USE_TURBOPACK=0

RUN npm run build

# =============================================
# Stage 3: Production Runner
# =============================================
FROM base AS runner

WORKDIR /app

ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1

# Create non-root user
RUN addgroup --system --gid 1001 nodejs \
    && adduser --system --uid 1001 nextjs

# Copy necessary files for production
COPY --from=builder /app/public ./public

# Copy standalone build output
COPY --from=builder --chown=nextjs:nodejs /app/.next/standalone ./
COPY --from=builder --chown=nextjs:nodejs /app/.next/static ./.next/static

USER nextjs

EXPOSE 3000

ENV PORT=3000
ENV HOSTNAME="0.0.0.0"

CMD ["node", "server.js"]