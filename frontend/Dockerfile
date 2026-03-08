FROM node:22-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
ARG VITE_BACKEND_URL=
ENV VITE_BACKEND_URL=$VITE_BACKEND_URL
RUN npm run build

FROM node:22-alpine
WORKDIR /app
COPY --from=builder /app/build ./build
COPY --from=builder /app/package*.json ./
RUN npm ci --omit=dev
ENV PORT=5173
ENV HOST=0.0.0.0
EXPOSE 5173
CMD ["node", "build"]
