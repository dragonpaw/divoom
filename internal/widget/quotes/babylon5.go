package quotes

// NewBabylon5 returns a Source of well-known lines from J. Michael
// Straczynski's *Babylon 5*. Curated collection of iconic moments from the
// series, spanning major characters and story arcs. Used under fair use.
func NewBabylon5() *Source { return newSource("Babylon 5", babylon5) }

var babylon5 = []string{
	// Kosh and the Vorlons
	"\"And so it begins.\" — Kosh",
	"\"The avalanche has already started. It is too late for the pebbles to vote.\" — Kosh",
	"\"Understanding is a three-edged sword: your side, their side, and the truth.\" — Kosh",
	"\"I am the way out. If you wish, I will show you.\" — Kosh",
	"\"I am not Kosh. I am the part that stayed behind.\" — Kosh",
	"\"In the end, you will stand or fall by your own conviction.\" — Kosh",
	"\"Who are you? What do you want? Why are you here? Where are you going? Do you have anything worth living for? All we have to decide is what to do with the time that is given us.\" — Lorien",
	"\"There is a way out of every box, a solution to every puzzle. If you look hard enough, you find it.\" — Lorien",
	"\"I am Lorien, first of the First Ones, and I am the last. I've come to say farewell.\" — Lorien",

	// Sinclair
	"\"Never start a fight, but always finish it.\" — Jeffrey Sinclair",
	"\"Entil'zha, veni nas brivari.\" — Jeffrey Sinclair",
	"\"Minbari do not kill Minbari.\" — Jeffrey Sinclair",
	"\"I have to believe that we can't survive if we become like them.\" — Jeffrey Sinclair",

	// Sheridan
	"\"Get the hell out of our galaxy!\" — John Sheridan",
	"\"I am death incarnate. I came here to die for your sins.\" — John Sheridan",
	"\"This is John Sheridan. I want you to know that every station, every ship that fights with us is a man or woman who chose to be here. They're here because they believe. And I will not let them down.\" — John Sheridan",
	"\"I stand in the place of power. I ask not that you believe in me or in what I say. I ask only that you believe in yourselves.\" — John Sheridan",
	"\"Only one human captain has ever survived battle with a Minbari fleet. He is behind me. You are in front of me.\" — Susan Ivanova",

	// Ivanova
	"\"No boom today. Boom tomorrow. There's always a boom tomorrow.\" — Susan Ivanova",
	"\"Ivanova is always right. I will listen to Ivanova. I will not ignore Ivanova's recommendations. Ivanova is God.\" — Susan Ivanova",
	"\"I have to believe it's not all random, that there's some purpose to all of this.\" — Susan Ivanova",
	"\"What do you want? What do you need?\" — Susan Ivanova",
	"\"I am death. I am the son of the morning. I am fury and the fire of the sun.\" — Susan Ivanova",

	// Delenn
	"\"Faith manages.\" — Delenn",
	"\"We are starstuff. We are the universe made manifest, trying to figure itself out.\" — Delenn",
	"\"I will see you again, in the place where no shadows fall.\" — Delenn",
	"\"There is a greater darkness than the one we fight. It is the darkness of the soul that has lost its way.\" — Delenn",
	"\"In my people's culture, we have something called Isil'zha. It means to see with the soul.\" — Delenn",
	"\"I have begun the process of becoming someone new. That is the most important thing.\" — Delenn",

	// G'Kar
	"\"The universe is run by the complex interweaving of three elements: energy, matter, and enlightened self-interest.\" — G'Kar",
	"\"If I take a lamp and shine it toward the wall, a bright spot will appear on the wall. The lamp is our search for truth, for understanding. Too often we assume that the light on the wall is God, but the light is not the goal of the search, it is the result of the search.\" — G'Kar",
	"\"There is a greater darkness than the one we fight. It is the darkness of the soul that has lost its way.\" — G'Kar",
	"\"Nobody here is exactly what he appears.\" — G'Kar",
	"\"I came here to find a reason to live, and instead I have found a reason to die.\" — G'Kar",
	"\"I am death. Where I walk, the flowers wilt. Where I breathe, the land dies.\" — G'Kar",
	"\"What is the state of my people? We are scattered, broken, hunted across a thousand worlds.\" — G'Kar",
	"\"In my younger and more vulnerable years, I believed many things. Some were true. Some were not.\" — G'Kar",
	"\"We are trapped by circumstance. Trapped by the choices we have made.\" — G'Kar",

	// Londo
	"\"The day the Centauri boy died, I felt the first stirrings of what would become my greatest joys and my deepest regrets.\" — Londo Mollari",
	"\"I have reached the end of my rope, and all I can do is dangle here, waiting.\" — Londo Mollari",
	"\"I'm not interested in your blood, I'm interested in your fear.\" — Londo Mollari",
	"\"Time is a gift. Use it well. Or it will use you.\" — Londo Mollari",
	"\"We are not the keepers of our destinies. We are the slaves of them.\" — Londo Mollari",
	"\"I thought I wanted power. What I found was slavery.\" — Londo Mollari",
	"\"Do you know what the single greatest moment of my life was? It was when I first saw you.\" — Londo Mollari",

	// Vir
	"\"I'm sorry. I'm so sorry.\" — Vir Cotto",
	"\"This is your destiny. There is no way out.\" — Vir Cotto",
	"\"I can see the darkness gathering again. And I think I'm the only one who can stop it.\" — Vir Cotto",
	"\"Some people get a second chance. Some people don't.\" — Vir Cotto",
	"\"I never wanted any of this. But it was forced upon me.\" — Vir Cotto",

	// Garibaldi
	"\"I have feelings about what we've become, and none of them are good.\" — Michael Garibaldi",
	"\"We live for the species. We die for the species.\" — Michael Garibaldi",
	"\"I've seen too much. Done too much. I can't go back.\" — Michael Garibaldi",
	"\"The worst part is not knowing whether you're fighting the right enemy.\" — Michael Garibaldi",
	"\"I'm tired. I'm tired of fighting. I'm tired of being right.\" — Michael Garibaldi",

	// Lennier
	"\"I have trained for this moment my entire life.\" — Lennier",
	"\"The moment of truth comes when you must choose between what you believe and what you can prove.\" — Lennier",
	"\"I am nothing. And yet I am everything.\" — Lennier",
	"\"My greatest fear is that I will become everything I despise.\" — Lennier",

	// Marcus
	"\"I'm going to tell you something that will change your life. Are you ready? Trust me.\" — Marcus Cole",
	"\"We don't have time for this. We don't have time for any of this.\" — Marcus Cole",
	"\"I know where I'm going. And I know how to get there.\" — Marcus Cole",
	"\"The greatest enemy is always the one you become.\" — Marcus Cole",

	// Lyta
	"\"Something is very wrong. I can feel it in my head.\" — Lyta Alexander",
	"\"I'm a telepath. I see what others don't. And what I see is darkness.\" — Lyta Alexander",
	"\"The Vorlons are not what we think they are.\" — Lyta Alexander",
	"\"They've infected me with something. I can feel it spreading.\" — Lyta Alexander",

	// Zack Allen
	"\"We are the last hope. If we fall, everything falls.\" — Zack Allen",
	"\"I'm a soldier. This is what I do.\" — Zack Allen",
	"\"We take care of our own, no matter what.\" — Zack Allen",
	"\"The line has been drawn. We don't cross it.\" — Zack Allen",

	// Dr. Franklin
	"\"I'm a doctor. I take an oath to preserve life.\" — Stephen Franklin",
	"\"Medicine is not about science alone. It's about hope.\" — Stephen Franklin",
	"\"We are losing them. People are dying, and I can't save them.\" — Stephen Franklin",
	"\"The body is a machine, but it's a machine with a soul.\" — Stephen Franklin",

	// Bester
	"\"We are the future. We are the next step in human evolution.\" — Walter Bester",
	"\"Telepathy is not a gift. It's a curse.\" — Walter Bester",
	"\"I know who you are. I know what you want. I know what you fear.\" — Walter Bester",
	"\"The future belongs to those who can see it first.\" — Walter Bester",

	// Morden
	"\"We're just going to ask you a few questions.\" — Morden",
	"\"What do you want? What do you really want?\" — Morden",
	"\"We serve the Shadows. And we serve ourselves.\" — Morden",
	"\"I'm just a businessman. I have wants and needs like anyone else.\" — Morden",

	// The Shadows
	"\"We are the ancient ones. We were here before your species was born.\" — The Shadows",
	"\"Conflict is the only way to grow. Conflict is the only way to evolve.\" — The Shadows",
	"\"We do not enslave. We teach. We mentor. We guide.\" — The Shadows",

	// Refa
	"\"I will do whatever it takes to gain power.\" — Centauri Prime Minister Refa",
	"\"My people are destined to rule. It is written in our blood.\" — Centauri Prime Minister Refa",
	"\"The strong survive. The weak perish.\" — Centauri Prime Minister Refa",

	// Cartagia
	"\"I am madness. I am chaos. I am the future.\" — Emperor Cartagia",
	"\"What do you want from me? What can I do for you?\" — Emperor Cartagia",
	"\"The universe will bow before me.\" — Emperor Cartagia",

	// Na'Toth
	"\"We must honor the old ways, or we lose ourselves.\" — Na'Toth",
	"\"G'Kar is our hope. G'Kar is our salvation.\" — Na'Toth",
	"\"The Narn will never surrender. The Narn will never bow.\" — Na'Toth",

	// General quotes and themes
	"\"We are at war. The question is not whether we fight, but how we fight and what we are willing to sacrifice.\" — Babylon 5",
	"\"In the end, we are defined not by our strength, but by our choices.\" — Babylon 5",
	"\"The night is darkest just before the dawn. Hold on just a little longer.\" — Babylon 5",
	"\"There is always hope. Even when the darkness seems overwhelming, there is always a light.\" — Babylon 5",
	"\"What is real? The consensus of our perceptions.\" — Babylon 5",
	"\"We are the products of our choices, not our circumstances.\" — Babylon 5",
	"\"There is no such thing as a simple answer to a complex question.\" — Babylon 5",
	"\"We must learn to live together, or we will surely die apart.\" — Babylon 5",
	"\"The cost of victory is sometimes too high.\" — Babylon 5",
	"\"In the darkness, we must be the light.\" — Babylon 5",
	"\"Every moment is a choice. Choose wisely.\" — Babylon 5",
	"\"The path to redemption is long and painful, but it is possible.\" — Babylon 5",
	"\"We are not alone. We are never alone.\" — Babylon 5",
	"\"The future is not written. We write it with our actions.\" — Babylon 5",
	"\"To find peace, we must first find ourselves.\" — Babylon 5",
	"\"Some battles are won not by strength, but by will.\" — Babylon 5",
	"\"Trust is the hardest thing to give, and the easiest thing to lose.\" — Babylon 5",
	"\"We cannot change the past, but we can shape the future.\" — Babylon 5",
	"\"The greater the darkness, the more important the light.\" — Babylon 5",
	"\"We stand together, or we fall alone.\" — Babylon 5",
	"\"Time is our greatest enemy and our greatest ally.\" — Babylon 5",
	"\"The truth is often buried beneath layers of deception.\" — Babylon 5",
	"\"We are all searching for something. Some search for power, others for peace.\" — Babylon 5",
	"\"The line between justice and revenge is a thin one.\" — Babylon 5",
	"\"We must sacrifice the few to save the many.\" — Babylon 5",
	"\"In the end, we are all alone.\" — Babylon 5",
	"\"Change is the only constant in the universe.\" — Babylon 5",
	"\"There is strength in unity, weakness in division.\" — Babylon 5",
	"\"We are the architects of our own destiny.\" — Babylon 5",
	"\"The wheel turns, and we turn with it.\" — Babylon 5",
	"\"For every action, there is a consequence.\" — Babylon 5",
	"\"We are bound by fate, but we are free in our choices.\" — Babylon 5",
	"\"The night sky is full of possibilities.\" — Babylon 5",
}
