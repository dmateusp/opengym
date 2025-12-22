# opengym 🏐

OpenGym is a web-app to organize and participate in group sports.

The name comes from the term "open gym" used in volleyball to refer to a casual practice session which isn't coach-led.

##  Background

I'm part of a few informal (unfederated) volleyball groups, and we currently use WhatsApp polls to organize games.

This is how it works:

* A poll is created with the place, date and time of the game.
* Members of the group can vote to join the game.
  * The game can be capped to a set number of players, the remaining vote into a waitlist.
  * The admin can allow players to bring guests from outside the group.
* When a minimum of players is reached, the organizer books the space.
* The poll is declared "closed" a day or a few days before the game.
* Total cost is divided among the players and payment is collected.
  * Payment can be collected before or after the game depending on the organizer.
  * The organizer can accept multiple payments methods.

WhatsApp polls are very accessible and easy to use, but they have limited features:

* You can't edit details of a poll, so if the date, time or place of the game changes, you have to communicate the new details to everyone and confirm their participation.
* Waitlists are by order of voting, and it can be difficult as a participant to follow your status in the list.
* It's hard to verify who paid and who didn't.
* You can't actually lock a poll.
* You can't track how participants change their voting, if they remove their vote etc.

##  Roadmap

* [ ] Admins and participants can log-in easily using OAuth.
* [ ] An admin can create a game and define basic details about the game.
* [ ] Participants can join a game, leave a game, and view the game details.

At this point we'd have equivalent functionality to a WhatsApp poll.

* [ ] Admins can enable a waitlist.
* [ ] Admins can lock a game.
* [ ] Admins can allow participants to bring guests. These participants are responsible for their guests' payment.
* [ ] Changes in participants' voting are tracked and shown.
* [ ] Payment total per participant is shown.
* [ ] Participants can signal that they paid, admins can confirm they received the payment.
* [ ] When admins change the game's important details (date, time, place, duration), the participants' votes are set to be confirmed again.
* [ ] Configurable payment links and methods are available.
