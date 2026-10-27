# Product

<!-- impeccable:product-schema 1 -->

## Platform

web

## Users

Tabletop RPG groups playing **online, each person at their own PC** (voice in Discord or similar, DogmaLiter open in a browser tab beside it). Game masters and players are **equally important**:

- **Game master** builds the campaign: invites players, authors the item compendium, hands out loot, watches the activity log and chat history, manages roles.
- **Player** lives in their character: sheet and attributes, grid inventory and equipment, trading items with other players, in-game chat.

Desktop is the primary viewport. Phones must remain usable (the project already has phone-responsive layouts) but are secondary.

## Product Purpose

A browser-based tabletop RPG management platform: campaigns, characters, grid inventories, item compendium, player-to-player trading, chat and party state in one place, with no downloads or self-hosting. Success is a session where nobody leaves the tab to track loot, stats or trades on paper or in spreadsheets.

## Positioning

A Diablo/Tarkov-style **spatial grid inventory** (multi-cell items, rotation, equipment slots, durability, enchantment, requirements that grey out gear) combined with a GM-curated compendium and escrowed player-to-player trading, built for system-agnostic tabletop play rather than a single ruleset.

## Operating Context

- Long sessions (hours) in the evening, dark rooms, alongside a voice call — the app sits on screen for the whole session.
- Session screen has tabs: Sheet, Inventory, Characters, Compendium, and a GM-only Manage panel (Players, Chat History, Activity tables).
- Drag-and-drop via interact.js is core to inventory, equipment, compendium delivery and trading.
- Public surfaces: landing page, news, plans (Free / paid tiers via Stripe), terms, privacy, auth.
- Admin area: users, news, games.

## Capabilities and Constraints

- Stack: Vue 3 + Vite, Vue Router, Pinia, Tailwind CSS 4, interact.js, Lucide icons, notivue toasts. Backend Go + MySQL.
- Attributes STR/DEX/CON/INT/WIS/CHA plus custom attributes; currency Gold/Silver/Bronze.
- Item rarity tiers exist and carry meaning (color-coded rarity is a domain convention, not decoration).
- Plan limits: games owned, players per game, maps, items, characters per game, upload size and storage.
- UI copy is English.

## Brand Commitments

- Name: **DogmaLiter**. Tagline in README: "Your Tabletop, Reimagined."
- Logo: `frontend/src/assets/DLlogo.png` — a d20 die whose faces frame a painted fantasy landscape (castle, mountains, dragons, campfire) in sky blue, periwinkle and deep navy, with orchid-pink moons and stars. It is the community's logo.
- **Binding:** the interface stays a **dark, blue theme** that matches the logo.
- Existing art assets: `frontend/src/assets/background1.jpg`, `background2.jpg` (landing), equipment-slot silhouettes (`armour.png`, `helmet.png`, `weapon.png`, `ring1.png`, etc.).

## Evidence on Hand

- Live instance: https://app.dogmaliter.online
- Feature screenshots in README.md (session, inventory, compendium, trading, chat, manage, admin).
- No testimonials, user counts, press or reviews exist — do not fabricate any.

## Product Principles

1. **The session tab is the product.** Every decision is judged by how it feels three hours into a game night.
2. **Players and GMs are peers.** Player surfaces get the same care as GM tooling.
3. **Game data carries the color.** Rarity, durability, requirements and trade state are what deserve emphasis; chrome stays out of their way.
4. **Spatial and tactile.** Inventory and equipment should feel like handling objects, not editing records.

## Accessibility & Inclusion

Long dark-room sessions: comfortable contrast without glare, readable small text in dense tables and item cards, rarity never conveyed by color alone.
