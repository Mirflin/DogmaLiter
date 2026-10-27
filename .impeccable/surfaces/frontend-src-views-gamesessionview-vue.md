---
version: 1
slug: "frontend-src-views-gamesessionview-vue"
primary_target: "frontend/src/views/GameSessionView.vue"
related_targets: ["frontend/src/views/DashboardView.vue"]
---

# Session & app shell — surface brief

Scope: the authenticated app (game session tabs, dashboard, games, profile, settings, admin, modals). Visitor mode: **Operate**. The public landing page is a separate Persuade surface and gets its own brief later.

Audience/job: players and GMs at their own PCs, hours-long evening sessions next to a voice call. Players handle their character sheet, grid inventory, equipment and trades; GMs author the compendium, deliver loot, read activity and chat history.

Constraints: dark, blue theme matching the d20 logo (binding). Keep all behaviour, copy and interact.js drag mechanics.

## Direction contract

THESIS: The interface is a dice tray. Work areas are recessed wells in deep navy velvet; everything a player holds is translucent coloured resin. Refuses the category default of raised slate cards with drop shadows and one blue accent.

OWN-WORLD: One 11-step navy ramp is the only neutral (raise from the darkroom record). Panels are inset wells, not elevated cards. Saturated colour lives only on data: rarity resin colourways, durability, trade/escrow state, stat modifiers (raise from the transit diagram). Periwinkle primary, sky-blue links/rare, orchid-pink (logo moons) for modifiers and incoming trade. Workhorse grotesque for UI; a semi-condensed face for inked numerals.

STORY: A player opens Inventory, reads health, AC and coin at a glance, sees what each item is from its resin colour and detail, drags gear into slots, and answers a trade without leaving the tab.

FIRST VIEWPORT: Top bar: logo mark, campaign name, tabs, session meta. Three columns: left well with portrait, Health/AC numerals, 3x2 attribute grid, equipment slots; centre coin row and the inventory grid as the tray floor with resin item tiles, trade offer bar beneath; right well with party chat and composer. Primary action is the trade Accept.

FORM: Polyhedral dice in a velvet tray, candidate 4 of 7 on the ordered list. seed:4916f2a3bd8f478f. Signature move: dragging an item dims every cell and slot that cannot accept it (raise from the streaming wall); item importance shown by how much apparatus it carries, not size (raise from the lexicon).

FINISH: unreviewed and undocumented is unfinished; this build ends with the finish review, the verdict, DESIGN.md, and every shipping raster carrying its provenance
