# opengym 🏐

opengym is a web-app to organize and participate in group sports.

The name comes from the term "open gym" used in volleyball to refer to a casual practice session which isn't coach-led.

## Getting Started

You can start running a "demo" version of opengym locally by running the following command:

```bash
go run cmd/opengymserver/main.go -demo.enabled=true -demo.auth.signing-secret=$(openssl rand -hex 32)
```

Then, start the front-end:

```bash
cd frontend
pnpm run dev
```

The demo version allows you to try out the various features of opengym both as a participant and as an organizer.

Once started, access the front-end and click on the profile icon in the top-right corner to impersonate any of the demo users.

## How to Create a Game

Any participant can create a game. Initially, only a name is required. The game is created in "draft" mode, meaning it is not yet visible to other participants.

### Game Configuration Options

- **Name:** Short title for the game.
- **Location:** Where the game will take place.
- **When:** Date and time the game will start.
- **Duration:** Length of the game.
- **Price:** Total cost incurred by the organizer (can be free).
  - The price is divided among participants. Each participant is shown an equal share, unless they are bringing guests, in which case they are responsible for their guests' share as well.
  - Note: Payment is _not handled by opengym_, but opengym helps organizers keep track of who has paid.
- **Participants:** Maximum number of participants who can join before the game is full.
- **Waitlist:** Number of participants who can be put on a waitlist if the game is full (can be disabled or set to a specific number).

## Publishing a Game

Publishing a game is disabled until the organizer has set all of the important information (details that would directly influence a participant's decision to join).
Once all required information has been set, the organizer can either publish the game immediately or schedule it for publishing at a later date and time.

Scheduling a game's publication gives the organizer a chance to notify potential participants about when a game will be available to join, making it fairer in cases where spots are limited.

Once a game is published, anyone with the link can join.

Note: opengym does not currently make games "searchable" on the platform. It is designed for small private communities/groups to organize games with their existing members, rather than providing features to recruit new members.

opengym tracks the date-time a player voted to join a game, and uses this information to determine the order in which players are added to the game.

## Participating in a Game (First Come, First Served)

A participant can vote to join a game, and later change their vote as their availability changes.

opengym tracks the date and time a participant voted to join a game, and uses this information to determine the order in which participants are added to the game.

- The organizer always has priority to join their own game.
  - The organizer does not have to join the game.
  - If the organizer joins, they will be added as the first participant, potentially pushing the last participant to the waitlist.
- Participants can vote to join the game until both the game and the waitlist are full.
- If a participant decides to leave the game, their vote is moved to a "Not going" section.
  - If that participant was going to the game, the first participant in the waitlist will take their spot.
  - If that participant decides to join the game again, they will be added to the bottom of the waitlist.

## Contributing

To suggest a new feature, open a [GitHub discussion](https://github.com/dmateusp/opengym/discussions). When we've discussed the feature and decided to implement it, we'll create a GitHub issue.

Not all contributions are code—we also welcome documentation, translations, and general feedback or ideas.

The [AGENTS.md](AGENTS.md) file describes the tooling we use for development; it should contain everything you need to know to start contributing code.

## Deployment

⚠️ Coming soon.
