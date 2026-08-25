# syntax=docker/dockerfile:1
FROM node:24-alpine AS deps
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci

FROM node:24-alpine AS build
WORKDIR /app
# NEXT_PUBLIC_* vars are inlined into the client bundle at build time, not
# read at container startup -- so this has to be a build ARG, not a runtime
# environment: value on the `web` service below.
ARG NEXT_PUBLIC_API_ORIGIN=http://localhost:8080
ENV NEXT_PUBLIC_API_ORIGIN=$NEXT_PUBLIC_API_ORIGIN
COPY --from=deps /app/node_modules ./node_modules
COPY . .
RUN npm run build

# output: 'standalone' in next.config.ts traces only the files this image
# actually needs, so the final stage never installs node_modules itself.
FROM node:24-alpine AS run
WORKDIR /app
ENV NODE_ENV=production
RUN addgroup -S hootden && adduser -S hootden -G hootden
COPY --from=build /app/public ./public
COPY --from=build --chown=hootden:hootden /app/.next/standalone ./
COPY --from=build --chown=hootden:hootden /app/.next/static ./.next/static
USER hootden
EXPOSE 3000
ENV PORT=3000
ENV HOSTNAME=0.0.0.0
CMD ["node", "server.js"]
