package namegen

import "math/rand"

var (
	adjectives = [...]string{
		"admiring",
		"adoring",
		"affectionate",
		"agitated",
		"amazing",
		"angry",
		"awesome",
		"beautiful",
		"blissful",
		"bold",
		"boring",
		"brave",
		"busy",
		"charming",
		"clever",
		"compassionate",
		"competent",
		"condescending",
		"confident",
		"cool",
		"cranky",
		"crazy",
		"dazzling",
		"determined",
		"distracted",
		"dreamy",
		"eager",
		"ecstatic",
		"elastic",
		"elated",
		"elegant",
		"eloquent",
		"epic",
		"exciting",
		"fervent",
		"festive",
		"flamboyant",
		"focused",
		"friendly",
		"frosty",
		"funny",
		"gallant",
		"gifted",
		"goofy",
		"gracious",
		"great",
		"happy",
		"hardcore",
		"heuristic",
		"hopeful",
		"hungry",
		"infallible",
		"inspiring",
		"intelligent",
		"interesting",
		"jolly",
		"jovial",
		"keen",
		"kind",
		"laughing",
		"loving",
		"lucid",
		"magical",
		"modest",
		"musing",
		"mystifying",
		"naughty",
		"nervous",
		"nice",
		"nifty",
		"nostalgic",
		"objective",
		"optimistic",
		"peaceful",
		"pedantic",
		"pensive",
		"practical",
		"priceless",
		"quirky",
		"quizzical",
		"recursing",
		"relaxed",
		"reverent",
		"romantic",
		"sad",
		"serene",
		"sharp",
		"silly",
		"sleepy",
		"stoic",
		"strange",
		"stupefied",
		"suspicious",
		"sweet",
		"tender",
		"thirsty",
		"trusting",
		"unruffled",
		"upbeat",
		"vibrant",
		"vigilant",
		"vigorous",
		"wizardly",
		"wonderful",
		"xenodochial",
		"youthful",
		"zealous",
		"zen",
	}

	//nolint:lll // too long descriptions
	surnames = [...]string{
		"ada",          // Ada Lovelace invented the first algorithm. https://en.wikipedia.org/wiki/Ada_Lovelace (thanks James Turnbull)
		"archimedes",   // Archimedes was a physicist, engineer and mathematician who invented too many things to list them here. https://en.wikipedia.org/wiki/Archimedes
		"banach",       // Stefan Banach - Polish mathematician, was one of the founders of modern functional analysis. https://en.wikipedia.org/wiki/Stefan_Banach
		"bell",         // Alexander Graham Bell - an eminent Scottish-born scientist, inventor, engineer and innovator who is credited with inventing the first practical telephone - https://en.wikipedia.org/wiki/Alexander_Graham_Bell
		"benz",         // Karl Friedrich Benz - a German automobile engineer. Inventor of the first practical motorcar. https://en.wikipedia.org/wiki/Karl_Benz
		"bohr",         // Niels Bohr is the father of quantum theory. https://en.wikipedia.org/wiki/Niels_Bohr.
		"chebyshev",    // Pafnuty Chebyshev - Russian mathematician. He is known for his works on probability, statistics, mechanics, analytical geometry and number theory https://en.wikipedia.org/wiki/Pafnuty_Chebyshev
		"cohen",        // Bram Cohen - American computer programmer and author of the BitTorrent peer-to-peer protocol. https://en.wikipedia.org/wiki/Bram_Cohen
		"curie",        // Marie Curie discovered radioactivity. https://en.wikipedia.org/wiki/Marie_Curie.
		"darwin",       // Charles Darwin established the principles of natural evolution. https://en.wikipedia.org/wiki/Charles_Darwin.
		"davinci",      // Leonardo Da Vinci invented too many things to list here. https://en.wikipedia.org/wiki/Leonardo_da_Vinci.
		"diffie",       // Bailey Whitfield Diffie - American cryptographer and one of the pioneers of public-key cryptography. https://en.wikipedia.org/wiki/Whitfield_Diffie
		"dijkstra",     // Edsger Wybe Dijkstra was a Dutch computer scientist and mathematical scientist. https://en.wikipedia.org/wiki/Edsger_W._Dijkstra.
		"dirac",        // Paul Adrien Maurice Dirac - English theoretical physicist who made fundamental contributions to the early development of both quantum mechanics and quantum electrodynamics. https://en.wikipedia.org/wiki/Paul_Dirac
		"einstein",     // Albert Einstein invented the general theory of relativity. https://en.wikipedia.org/wiki/Albert_Einstein
		"elgamal",      // Taher A. ElGamal - Egyptian cryptographer best known for the ElGamal discrete log cryptosystem and the ElGamal digital signature scheme. https://en.wikipedia.org/wiki/Taher_Elgamal
		"euclid",       // Euclid invented geometry. https://en.wikipedia.org/wiki/Euclid
		"euler",        // Leonhard Euler invented large parts of modern mathematics. https://de.wikipedia.org/wiki/Leonhard_Euler
		"faraday",      // Michael Faraday - British scientist who contributed to the study of electromagnetism and electrochemistry. https://en.wikipedia.org/wiki/Michael_Faraday
		"feistel",      // Horst Feistel - German-born American cryptographer who was one of the earliest non-government researchers to study the design and theory of block ciphers. Co-developer of DES and Lucifer. Feistel networks, a symmetric structure used in the construction of block ciphers are named after him. https://en.wikipedia.org/wiki/Horst_Feistel
		"fermat",       // Pierre de Fermat pioneered several aspects of modern mathematics. https://en.wikipedia.org/wiki/Pierre_de_Fermat
		"feynman",      // Richard Feynman was a key contributor to quantum mechanics and particle physics. https://en.wikipedia.org/wiki/Richard_Feynman
		"gagarin",      // Yuri Alekseyevich Gagarin - Soviet pilot and cosmonaut, best known as the first human to journey into outer space. https://en.wikipedia.org/wiki/Yuri_Gagarin
		"galileo",      // Galileo was a founding father of modern astronomy, and faced politics and obscurantism to establish scientific truth.  https://en.wikipedia.org/wiki/Galileo_Galilei
		"galois",       // Évariste Galois - French mathematician whose work laid the foundations of Galois theory and group theory, two major branches of abstract algebra, and the subfield of Galois connections, all while still in his late teens. https://en.wikipedia.org/wiki/%C3%89variste_Galois
		"gauss",        // Johann Carl Friedrich Gauss - German mathematician who made significant contributions to many fields, including number theory, algebra, statistics, analysis, differential geometry, geodesy, geophysics, mechanics, electrostatics, magnetic fields, astronomy, matrix theory, and optics. https://en.wikipedia.org/wiki/Carl_Friedrich_Gauss
		"grothendieck", // Alexander Grothendieck - German-born French mathematician who became a leading figure in the creation of modern algebraic geometry. https://en.wikipedia.org/wiki/Alexander_Grothendieck
		"hamilton",     // Margaret Hamilton - Director of the Software Engineering Division of the MIT Instrumentation Laboratory, which developed on-board flight software for the Apollo space program. https://en.wikipedia.org/wiki/Margaret_Hamilton_(scientist)
		"hawking",      // Stephen Hawking pioneered the field of cosmology by combining general relativity and quantum mechanics. https://en.wikipedia.org/wiki/Stephen_Hawking
		"heisenberg",   // Werner Heisenberg was a founding father of quantum mechanics. https://en.wikipedia.org/wiki/Werner_Heisenberg
		"hellman",      // Martin Edward Hellman - American cryptologist, best known for his invention of public-key cryptography in co-operation with Whitfield Diffie and Ralph Merkle. https://en.wikipedia.org/wiki/Martin_Hellman
		"hertz",        // Heinrich Rudolf Hertz - German physicist who first conclusively proved the existence of the electromagnetic waves. https://en.wikipedia.org/wiki/Heinrich_Hertz
		"kepler",       // Johannes Kepler, German astronomer known for his three laws of planetary motion - https://en.wikipedia.org/wiki/Johannes_Kepler
		"knuth",        // Donald Knuth - American computer scientist, author of "The Art of Computer Programming" and creator of the TeX typesetting system. https://en.wikipedia.org/wiki/Donald_Knuth
		"lamport",      // Leslie B. Lamport - American computer scientist. Lamport is best known for his seminal work in distributed systems and was the winner of the 2013 Turing Award. https://en.wikipedia.org/wiki/Leslie_Lamport
		"maxwell",      // James Clerk Maxwell - Scottish physicist, best known for his formulation of electromagnetic theory. https://en.wikipedia.org/wiki/James_Clerk_Maxwell
		"mccarthy",     // John McCarthy invented LISP: https://en.wikipedia.org/wiki/John_McCarthy_(computer_scientist)
		"mendel",       // Gregor Johann Mendel - Czech scientist and founder of genetics. https://en.wikipedia.org/wiki/Gregor_Mendel
		"mendeleev",    // Dmitri Mendeleev - a chemist and inventor. He formulated the Periodic Law, created a farsighted version of the periodic table of elements, and used it to correct the properties of some already discovered elements and also to predict the properties of eight elements yet to be discovered. https://en.wikipedia.org/wiki/Dmitri_Mendeleev
		"morse",        // Samuel Morse - contributed to the invention of a single-wire telegraph system based on European telegraphs and was a co-developer of the Morse code - https://en.wikipedia.org/wiki/Samuel_Morse
		"nash",         // John Forbes Nash, Jr. - American mathematician who made fundamental contributions to game theory, differential geometry, and the study of partial differential equations. https://en.wikipedia.org/wiki/John_Forbes_Nash_Jr.
		"neumann",      // John von Neumann - todays computer architectures are based on the von Neumann architecture. https://en.wikipedia.org/wiki/Von_Neumann_architecture
		"newton",       // Isaac Newton invented classic mechanics and modern optics. https://en.wikipedia.org/wiki/Isaac_Newton
		"noether",      // Emmy Noether, German mathematician. Noether's Theorem is named after her. https://en.wikipedia.org/wiki/Emmy_Noether
		"pascal",       // Blaise Pascal, French mathematician, physicist, and inventor - https://en.wikipedia.org/wiki/Blaise_Pascal
		"pike",         // Rob Pike was a key contributor to Unix, Plan 9, the X graphic system, utf-8, and the Go programming language. https://en.wikipedia.org/wiki/Rob_Pike
		"poincare",     // Henri Poincaré made fundamental contributions in several fields of mathematics. https://en.wikipedia.org/wiki/Henri_Poincar%C3%A9
		"ptolemy",      // Claudius Ptolemy - a Greco-Egyptian writer of Alexandria, known as a mathematician, astronomer, geographer, astrologer, and poet of a single epigram in the Greek Anthology - https://en.wikipedia.org/wiki/Ptolemy
		"ramanujan",    // Srinivasa Ramanujan - Indian mathematician and autodidact who made extraordinary contributions to mathematical analysis, number theory, infinite series, and continued fractions. - https://en.wikipedia.org/wiki/Srinivasa_Ramanujan
		"ritchie",      // Dennis Ritchie - co-creator of UNIX and the C programming language. - https://en.wikipedia.org/wiki/Dennis_Ritchie
		"satoshi",      // Satoshi Nakamoto is the name used by the unknown person or group of people who developed bitcoin, authored the bitcoin white paper, and created and deployed bitcoin's original reference implementation. https://en.wikipedia.org/wiki/Satoshi_Nakamoto
		"shamir",       // Adi Shamir - Israeli cryptographer whose numerous inventions and contributions to cryptography include the Ferge Fiat Shamir identification scheme, the Rivest Shamir Adleman (RSA) public-key cryptosystem, the Shamir's secret sharing scheme, the breaking of the Merkle-Hellman cryptosystem, the TWINKLE and TWIRL factoring devices and the discovery of differential cryptanalysis (with Eli Biham). https://en.wikipedia.org/wiki/Adi_Shamir
		"shannon",      // Claude Shannon - The father of information theory and founder of digital circuit design theory. (https://en.wikipedia.org/wiki/Claude_Shannon)
		"stallman",     // Richard Matthew Stallman - the founder of the Free Software movement, the GNU project, the Free Software Foundation, and the League for Programming Freedom. He also invented the concept of copyleft to protect the ideals of this movement, and enshrined this concept in the widely-used GPL (General Public License) for software. https://en.wikiquote.org/wiki/Richard_Stallman
		"tesla",        // Nikola Tesla invented the AC electric system and every gadget ever used by a James Bond villain. https://en.wikipedia.org/wiki/Nikola_Tesla
		"thompson",     // Ken Thompson - co-creator of UNIX and the C programming language - https://en.wikipedia.org/wiki/Ken_Thompson
		"torvalds",     // Linus Torvalds invented Linux and Git. https://en.wikipedia.org/wiki/Linus_Torvalds
		"turing",       // Alan Turing was a founding father of computer science. https://en.wikipedia.org/wiki/Alan_Turing.
		"wright",       // The Wright brothers, Orville and Wilbur - credited with inventing and building the world's first successful airplane and making the first controlled, powered and sustained heavier-than-air human flight - https://en.wikipedia.org/wiki/Wright_brothers

		// rofls
		"satont",     // author of https://github.com/Satont/tsuwari
		"rprtr",      // author of https://github.com/rprtr258/pm
		"coopertino", // C++ developer

		// cats
		"uuh",
		"plink",
		"vahui",
	}
)

// New generates a random name from the list of adjectives and surnames in this package
// formatted as "adjective-surname". For example "focused-turing".
//
//nolint:gosec // G404: Use of weak random number generator (math/rand instead of crypto/rand)
func New() string {
	adjective := adjectives[rand.Intn(len(adjectives))]
	surname := surnames[rand.Intn(len(surnames))]
	return adjective + "-" + surname
}
