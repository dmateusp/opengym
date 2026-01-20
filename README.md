# opengym 🏐

opengym is a web-app to organize and participate in group sports.

The name comes from the term "open gym" used in volleyball to refer to a casual practice session which isn't coach-led.

## Getting started

You can start running a "demo" version of opengym locally by running:

```bash
go run cmd/opengymserver/main.go -demo.enabled=true -demo.auth.signing-secret=`openssl rand -hex 32`
```

and starting the front-end with:

```bash
cd frontend
pnpm run dev
```

The demo version allows you to try out the various features of opengym both as a player and as an organizer.

Once started, access the front-end and click on the profile on the top-right corner to impersonate any of the demo users.

###  Creating a game

Any user can create a game, initially they only need to provide a name for it.
The game is created in "draft" mode, which means it is not yet visible to other users.

A game has the following configurations:

- **Name:** a short title for the game.
- **Location:** where the game will take place.
- **When:** the date and time the game will start.
- **Duration:** the length of the game.
- **Price:** the total cost incurred by the organizer. (Can be free)
  - This price is divided among the players. Each player is shown an equal share of the price, unless they are bringing guests, in which case they are responsible for their guests' share too.
  - Note that payment is *not handled by opengym*. But opengym helps organizers keep track of who paid and who did not.
- **Players:** how many players can join the game before it's full.
- **Waitlist:** how many players can be put in a waitlist if the game is full.
  - This can be disabled, or a specific number.

### Publishing a game

Publishing a game is disabled until the organizer has set all of the important information of the game (information that would directly influence a player's decision to join).
Once all of the required information has been set, the organizer can either publish the game right away or schedule it for publishing at a later date/time.
Scheduling a game's publication gives the organizer a chance to tell potential participants about when a game will be available to join, making it fairer in cases where spots are limited.

Once a game is published, anyone with the link can join the game.

Note that opengym does not currently make games "searchable" on the platform. In its current version, it is made for small private communities/groups to organize games with their existing members, rather than providing features to allow communities to recruit new members.

###  Participating in a game - first come first served

A player can vote to join a game, and later flip his/her vote as their availability changes.

opengym tracks the date-time a player voted to join a game, and uses this information to determine the order in which players are added to the game.

- An organizer always has priority to join his/her game.
  - The organizer doesn't have to join a game.
  - If the organizer joins the game, they will be added as the first player, potentially pushing the last player to the waitlist.
- Players can vote to join the game until both the game and the waitlist are full.
- If a player decides to leave the game, their vote is moved to a "Not going" section.
  - If that player was going to the game, the first player in the waitlist will takes his/her spot.
  - If that player decides to join the game again, they will be added to the bottom of the waitlist.

## Contributing

To suggest a new feature, open a GitHub discussion. When we've discussed the feature and decided to implement it, we'll create a GitHub issue.

Not all contributions are code, we also welcome documentation, translations, and general feedback/ideas.

The [AGENTS.md](./AGENTS.md) file describes the tooling we use for development, it should contain everything you need to know to start contributing code.

## Deployment

⚠️ Coming soon.
