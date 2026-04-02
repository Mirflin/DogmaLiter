# DogmaLiter

**Your Tabletop, Reimagined.**

DogmaLiter is a browser-based tabletop RPG management platform. Create campaigns, manage characters, organize inventories, track equipment, and play collaborative sessions online.

## Features

### Game Management
- Create campaigns with customizable settings
- Invite system with time-limited codes
- Roles: Game Master (GM) and Players
- Cover and map uploads

### Character System
- Create characters with backstories and portraits
- Base D&D 5e attributes (Strength, Dexterity, Constitution, Intelligence, Wisdom, Charisma)
- Custom attributes
- Currency tracking (Gold, Silver, Copper)

### Inventory & Equipment
- Grid-based inventory system (like classic RPG games)
- Equipment slots
- Item trading between characters

### Item System
- Rarity levels: common, uncommon, rare, epic, legendary, artifact
- Attribute modifiers and item requirements
- Multi-cell items in inventory

### In-Game Communication
- Chat with text and item links
- System messages

### Maps
- Upload custom maps with configurable grids
- Active map management

### News
- See news and updates

### Subscription Plans
| | Free | Plus ($4.99/mo) | Pro ($9.99/mo) |
|---|---|---|---|
| Games | 2 | 10 | Unlimited |
| Players/game | 5 | 15 | Unlimited |
| Maps/game | 3 | 15 | Unlimited |
| Items | 20 | 100 | Unlimited |
| Characters/game | 5 | 20 | Unlimited |
| Upload size | 5 MB | 25 MB | 50 MB |
| Storage | 100 MB | 1 GB | 5 GB |

## Tech Stack

### Backend
- **Go** 1.25 — server language
- **Chi** v5 — HTTP router
- **GORM** — ORM for MySQL
- **JWT** — authentication with refresh tokens
- **Stripe** — payment processing
- **GoMail** — email delivery

### Frontend
- **Vue 3** — UI framework
- **Vite** — build tool
- **Vue Router** — routing
- **Pinia** — state management
- **Tailwind CSS** — styling
- **Axios** — HTTP client

### Database
- **MySQL** 8.x

## Getting Started

### Prerequisites
- Go 1.25+
- Node.js 20.19+ or 22.12+
- MySQL 8.x

### Environment Setup

Create a `.env` file in the `/` directory:

```ini
# Database
DATABASE_URL=user:password@tcp(localhost:3306)/DogmaLiter?charset=utf8mb4&parseTime=True&loc=Local

# Server
PORT=8006
FRONTEND_URL=http://localhost:5175

# JWT
JWT_SECRET=your_jwt_secret_key

# SMTP
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=your_email@gmail.com
SMTP_PASSWORD=your_app_password
SMTP_FROM=noreply@dogmaliter.com

# File Uploads
UPLOAD_DIR=./uploads

# Stripe
STRIPE_SECRET_KEY=sk_...
STRIPE_WEBHOOK_KEY=whsec_...
STRIPE_PLUS_PRICE_ID=price_...
STRIPE_PRO_PRICE_ID=price_...
```

Create a `.env` file in the `frontend/` directory:

# Server
VITE_API_URL=localhost:8006

### Running the Backend

```bash
cd backend
go run cmd/server/main.go
```

### Running the Frontend

```bash
cd frontend
npm install
npm run dev
```

The app will be available at `http://localhost:5175`

### Production Build

```bash
# Backend
cd backend
go build -o dogmaliter ./cmd/server/main.go

# Frontend
cd frontend
npm run build
```

## 📁 Project Structure

```
DogmaLiter/
├── backend/
│   ├── cmd/server/          # Server entry point
│   ├── internal/
│   │   ├── auth/            # Authentication & authorization
│   │   ├── character/       # Character management
│   │   ├── config/          # App configuration
│   │   ├── game/            # Game & session logic
│   │   ├── inventory/       # Inventory system
│   │   ├── item/            # Item system
│   │   ├── models/          # Data models (GORM)
│   │   ├── news/            # News publishing
│   │   └── payment/         # Stripe integration
│   ├── pkg/
│   │   └── database/        # DB connection & migrations
│   └── uploads/             # Uploaded files
├── frontend/
│   ├── public/              # Static assets
│   └── src/
│       ├── api/             # HTTP client (Axios)
│       ├── assets/          # Styles
│       ├── components/      # Vue components
│       ├── layouts/         # Page layouts
│       ├── router/          # Routing
│       ├── stores/          # Pinia stores
│       └── views/           # App pages
```

## 📄 API

```
/api/
├── health                         GET
├── auth/
│   ├── register                   POST
│   ├── login                      POST
│   ├── refresh                    POST
│   ├── verify                     GET
│   ├── resend-verification        POST
│   ├── forgot-password            POST
│   └── reset-password             POST
├── games/
│   ├── /                          GET, POST
│   ├── /join                      POST
│   ├── /{gameID}                  GET, PUT, DELETE
│   ├── /{gameID}/play             GET
│   ├── /{gameID}/cover            POST
│   ├── /{gameID}/invite-code      GET
│   ├── /{gameID}/regenerate-code  POST
│   └── /leave                     POST
├── news/
│   ├── /                          GET, POST
│   └── /{id}                      GET
├── payment/
│   ├── /checkout                  POST
│   ├── /cancel                    POST
│   └── /webhook                   POST
├── plans                          GET
├── uploads/{id}                   GET
└── me                             GET
```