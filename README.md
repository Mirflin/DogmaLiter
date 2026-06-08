<h1 align="center">DogmaLiter</h1>

<p align="center"><b>Your Tabletop, Reimagined.</b></p>

<p align="center">
DogmaLiter is a browser-based tabletop RPG management platform. Create campaigns, manage
characters, organize grid inventories, trade items between players, run sessions, and keep
everything — chat, loot, and party state — in one place. No downloads, no hosting.
</p>

<p align="center">
  <a href="https://app.dogmaliter.online"><b>🌐 Live demo — app.dogmaliter.online</b></a>
</p>

<p align="center">
  <i>The project is live and can be reviewed online at
  <a href="https://app.dogmaliter.online">app.dogmaliter.online</a> — no setup required.</i>
</p>

<!-- ============================================================
     MAIN SHOWCASE GIF
     Drop a short product overview clip here (landing → session).
     Suggested path: docs/media/overview.gif
============================================================ -->
<p align="center">
  <img src="docs/media/overview.gif" alt="DogmaLiter overview" width="860" />
</p>

---

## ✨ Features

Each section below has a slot for a short GIF. Record the flow, drop the file at the
suggested path, and the preview will appear automatically.

### 🎲 Campaign & Session Management
Create campaigns with invite codes, manage members, and switch between Sheet, Inventory,
Characters, Compendium and the GM Manage panel. Per-game toggles control chat, standard
attributes, and item trading.

<!-- docs/media/campaign.gif — creating a game, inviting players, opening a session -->
<p align="center">
  <img src="docs/media/campaign.gif" alt="Campaign and session management" width="820" />
</p>

### 🧑 Characters
Characters with portraits and backstories, base attributes (STR/DEX/CON/INT/WIS/CHA),
custom attributes, and currency (Gold / Silver / Bronze). Effective stats update live from
equipped gear.

<!-- docs/media/character.gif — character sheet with attributes and currency -->
<p align="center">
  <img src="docs/media/character.gif" alt="Character sheet" width="820" />
</p>

### 🎒 Grid Inventory & Equipment
Drag-and-drop grid inventory with multi-cell items, equipment slots (weapons, armor, rings,
amulet), durability, enchantment, rotation, and split/unstack. Item requirements grey out
gear a character can't equip.

<!-- docs/media/inventory.gif — dragging items, equipping, splitting a stack -->
<p align="center">
  <img src="docs/media/inventory.gif" alt="Grid inventory and equipment" width="820" />
</p>

### 📦 Item Compendium
GMs build a shared compendium with rarity tiers, attribute modifiers, requirements, tags,
and images, then hand out loot to characters (add to a transfer cart by click or
double-click, then deliver).

<!-- docs/media/compendium.gif — creating an item and delivering it to a character -->
<p align="center">
  <img src="docs/media/compendium.gif" alt="Item compendium" width="820" />
</p>

### 🔁 Player-to-Player Trading
A player offers items from their character; the recipient accepts (if there's free space) or
declines. Offered items are held in escrow until the trade resolves. Searchable recipient
picker and item durability shown at selection.

<!-- docs/media/trading.gif — sending an offer and accepting/declining it -->
<p align="center">
  <img src="docs/media/trading.gif" alt="Player to player trading" width="820" />
</p>

### 💬 In-Game Chat
Text chat with shareable item links — click a shared item to inspect its full card. Can be
disabled per game.

<!-- docs/media/chat.gif — sending a message and sharing an item to chat -->
<p align="center">
  <img src="docs/media/chat.gif" alt="In-game chat" width="820" />
</p>

### 🛡️ GM Manage Panel
A GM-only control center with universal, searchable/filterable tables: **Players** (with role
management and a full-screen player overview), **Chat History**, and an **Activity** log that
records inventory and character actions (click item names to inspect). Tables auto-refresh and
can be cleared by time period. From the roster, GMs can inspect a character's sheet or open a
read-only view of their grid inventory.

<!-- docs/media/manage.gif — manage tab tables, player overview, inventory inspection -->
<p align="center">
  <img src="docs/media/manage.gif" alt="GM manage panel" width="820" />
</p>

### ⚙️ Admin Dashboard
Tabbed admin area for **Users**, **News**, and **Games**. Edit a user's role, subscription
plan and manual expiry, and verification status; publish/hide or edit news; remove games. All
deletions require confirmation.

<!-- docs/media/admin.gif — admin tabs, editing a subscription, publishing news -->
<p align="center">
  <img src="docs/media/admin.gif" alt="Admin dashboard" width="820" />
</p>

### 🗺️ Maps & 📰 News
Upload custom maps with configurable grids and manage the active map. A public news feed keeps
players up to date.

<!-- docs/media/maps-news.gif — uploading a map / browsing news -->
<p align="center">
  <img src="docs/media/maps-news.gif" alt="Maps and news" width="820" />
</p>

---

## 🧱 Tech Stack

**Backend** — Go 1.25, Chi v5 (router), GORM (MySQL ORM), JWT auth with refresh tokens,
Stripe (payments), GoMail (email).

**Frontend** — Vue 3, Vite, Vue Router, Pinia, Tailwind CSS, Axios, interact.js (inventory
drag-and-drop), Lucide icons.

**Database** — MySQL 8.x.

---

## 🚀 Getting Started

> 💡 **Just want to try it?** A hosted instance runs at **[app.dogmaliter.online](https://app.dogmaliter.online)** —
> no installation needed. The steps below are only for running it locally.

### Prerequisites
- Go 1.25+
- Node.js 20.19+ or 22.12+
- MySQL 8.x

### Environment Setup

Create a `.env` file in the project root:

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

```ini
VITE_API_URL=localhost:8006
```

### Running

```bash
# Backend
cd backend
go run cmd/server/main.go

# Frontend (in another terminal)
cd frontend
npm install
npm run dev
```

The app will be available at `http://localhost:5175`.

### Production Build

```bash
# Backend
cd backend
go build -o dogmaliter ./cmd/server/main.go

# Frontend
cd frontend
npm run build
```

---

## 📁 Project Structure

```
DogmaLiter/
├── backend/
│   ├── cmd/server/          # Server entry point
│   ├── internal/
│   │   ├── auth/            # Authentication & authorization
│   │   ├── config/          # App configuration
│   │   ├── game/            # Games, sessions, characters, inventory, items, trading, activity
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
│       ├── assets/          # Global styles
│       ├── components/      # Vue components (incl. session/* and DataTable)
│       ├── layouts/         # Page layouts
│       ├── router/          # Routing
│       ├── stores/          # Pinia stores
│       └── views/           # App pages
└── docs/
    └── media/               # README GIFs / screenshots
```

---

## 📄 API Overview

```
/api/
├── health                                          GET
├── me                                              GET
├── auth/  (register, login, refresh, verify, resend-verification,
│           forgot-password, reset-password)
├── games/
│   ├── /                                           GET, POST
│   ├── /join                                       POST
│   ├── /{gameID}                                   GET, PUT, DELETE
│   ├── /{gameID}/session                           GET
│   ├── /{gameID}/cover                             POST
│   ├── /{gameID}/invite-code | regenerate-code     GET | POST
│   ├── /{gameID}/leave                             POST
│   ├── /{gameID}/members/{userID}                  PATCH, DELETE
│   ├── /{gameID}/characters[/{characterID}]        GET, POST, PATCH, DELETE
│   ├── /{gameID}/characters/{id}/inventory[/{i}]   POST, PUT, PATCH, DELETE, split
│   ├── /{gameID}/characters/{id}/portrait          POST
│   ├── /{gameID}/items[/{itemID}]                  GET, POST, PATCH, DELETE, image
│   ├── /{gameID}/trades[/{tradeID}/accept|decline] GET, POST
│   ├── /{gameID}/activity                          GET, DELETE
│   └── /{gameID}/chat-messages                     GET, POST, DELETE
├── news/  /{id}                                    GET, POST, PATCH, DELETE
├── payment/  (checkout, cancel, webhook)           POST
├── plans                                           GET
├── uploads/{id}                                    GET
└── admin/  (stats, users, games, plans)            GET, PATCH, DELETE
```
