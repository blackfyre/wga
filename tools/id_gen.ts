import * as fs from 'fs';

type ArtistMetaInfo = {
	year_of_birth?: number | null;
	year_of_death?: number | null;
	place_of_birth?: string | null;
	place_of_death?: string | null;
	year_active_start?: number | null;
	year_active_end?: number | null;
	active_at_location?: string | null;
	exact_year_of_birth?: boolean;
	exact_year_of_death?: boolean;
};

type ArtistReferenceRecord = {
	id?: string;
	slug?: string;
	ARTIST: string;
	'BIRTH DATA': string;
	PROFESSION: string;
	SCHOOL: string;
	URL: string;
	wga_id?: string;
	meta?: ArtistMetaInfo | null;
};

const idTracker: string[] = [];
const ArtistLifePatters = {
	KnownBirthYearLocationKnownDeathYearLocation: /^\(b\.?\s(\d{4}),\s(.*),\sd\.?\s(\d{4}),\s(.*)\)$/,
	CircaBornYearKnownLocationKnownDeathYearLocation:
		/^\(b\.?\sca\.?\s(\d{4}),\s(.*),\sd\.?\s(\d{4}),\s(.*)\)$/,
	CircaBornYearKnownLocationKnownDeathYearLocation2:
		/^\(b\.?\s(\d{4})s,\s(.*),\sd\.?\s(\d{4}),\s(.*)\)$/,
	KnownBornYearLocationCircaDeathKnownLocation:
		/^\(b\.?\s(\d{4}),\s(.*),\sd\.?\s(?:ca\.?|after)\s?(\d{4}),\s(.*)\)$/,
	CircaBirthYearKnownLocationCircaDeathKnownLocation:
		/^\(b\.?\sca\.?\s(\d{4}),\s(.*),\sd\.?\sca\.?\s(\d{4}),\s(.*)\)$/,
	CircaBirthYearKnownLocationKnownDeathYearUnknownLocation:
		/^\(b\.?\sca\.?\s(\d{4}),\s(.*),\sd\.?\s(\d{4})\)$/,
	KnownToBeActiveInASingleLocation: /^\(active\s(\d{4})-(\d{2,4})\si?n?\s?(.*)\)$/,
	KnownToBeActiveAtInTimeNoLocation: /^\(.*(\d{4})-(\d{2,4})\)$/,
	KnwonDeathYearLocation: /^\(d\.\s?(\d{4}),\s?(.*)\)$/,
	KnownBirthYearRangeLocationKnownDeathYearRangeLocation:
		/^\(b\.?\s?(\d{4})\/(\d{2,4}),\s?(.*),\s?d\.?\s?(\d{4})\/(\d{2,4}),\s(.*)\)$/,
	KnownActiveYearLocationStartCircaDeathDeathYearKnownLocation:
		/^\(active\s?(\d{4}),\s?(.*),\s?d\.?\s?ca\.?\s?(\d{4}),\s?(.*)\)$/,
	KnownBirthYearLocationCircaDeathYearUnknownLocation:
		/^\(b\.?\s(\d{4}),\s(.*),\sd\.?\s(?:ca\.?|after)\s?(\d{4})\)$/
};

/**
 * It takes a string, removes all accents, lowercases it, removes all non-alphanumeric characters, and
 * replaces spaces with dashes
 * @param {(string | number)[]} args - (string | number)[]
 * @returns A function that takes a variable number of arguments and returns a string.
 */
export const slugify = (...args: (string | number)[]): string => {
	const value = args.join(' ');

	return value
		.normalize('NFD') // split an accented letter in the base letter and the acent
		.replace(/[\u0300-\u036f]/g, '') // remove all previously split accents
		.toLowerCase()
		.trim()
		.replace(/[^a-z0-9 ]/g, '') // remove all chars not letters, numbers and spaces (to be replaced)
		.replace(/\s+/g, '-'); // separator
};

/**
 * It takes an array of numbers, adds them all together, divides the sum by the length of the array,
 * and returns the result
 * @param {number[]} arr - number[] - The array of numbers to average.
 * @returns The average of the array.
 */
export const ArrayAverage = (arr: number[]): number => {
	return arr.reduce((p, c) => p + c, 0) / arr.length;
};

/**
 * "Given a year, extrapolate the second year of a two year range."
 *
 * The function takes two parameters:
 *
 * * `firstYear`: The first year of a two year range.
 * * `secondYear`: The second year of a two year range
 * @param {number} firstYear - The first year of the range.
 * @param {number} secondYear - The year you want to extrapolate to.
 * @returns A number
 */
export const ExtrapolateSecondYear = (firstYear: number, secondYear: number): number => {
	let firstPart = parseInt(firstYear.toString().slice(0, 2));
	let secondPart = parseInt(firstYear.toString().slice(2, 4));

	if (secondPart > secondYear) {
		firstPart++;
	}

	return parseInt(`${firstPart}${secondYear}`);
};

/**
 * Generate a random string of a given length.
 * @param {number} length - The length of the string you want to generate.
 * @returns A random string of characters.
 */
export function makeid(length: number) {
	var result = '';
	var characters = 'abcdefghijklmnopqrstuvwxyz0123456789';
	var charactersLength = characters.length;
	for (var i = 0; i < length; i++) {
		result += characters.charAt(Math.floor(Math.random() * charactersLength));
	}
	return result;
}

/* Regex for the old ID */
const regex = /https:\/\/www\.wga\.hu\/bio\/[a-z]\/(.*)\/biograph\.html/;

const generateMetaInfo = (birthData: string): ArtistMetaInfo | null => {
	let match = birthData.match(ArtistLifePatters.KnownBirthYearLocationKnownDeathYearLocation);
	if (match) {
		return {
			year_of_birth: Number(match[1]),
			place_of_birth: match[2],
			year_of_death: Number(match[3]),
			place_of_death: match[4],
			exact_year_of_birth: true,
			exact_year_of_death: true
		};
	}

	match = birthData.match(ArtistLifePatters.CircaBornYearKnownLocationKnownDeathYearLocation);
	if (match) {
		return {
			year_of_birth: Number(match[1]),
			place_of_birth: match[2],
			year_of_death: Number(match[3]),
			place_of_death: match[4],
			exact_year_of_birth: false,
			exact_year_of_death: true
		};
	}

	match = birthData.match(ArtistLifePatters.CircaBornYearKnownLocationKnownDeathYearLocation2);
	if (match) {
		return {
			year_of_birth: Number(match[1]),
			place_of_birth: match[2],
			year_of_death: Number(match[3]),
			place_of_death: match[4],
			exact_year_of_birth: false,
			exact_year_of_death: true
		};
	}

	match = birthData.match(ArtistLifePatters.KnownBornYearLocationCircaDeathKnownLocation);
	if (match) {
		return {
			year_of_birth: Number(match[1]),
			place_of_birth: match[2],
			year_of_death: Number(match[3]),
			place_of_death: match[4],
			exact_year_of_birth: true,
			exact_year_of_death: false
		};
	}

	match = birthData.match(ArtistLifePatters.CircaBirthYearKnownLocationCircaDeathKnownLocation);
	if (match) {
		return {
			year_of_birth: Number(match[1]),
			place_of_birth: match[2],
			year_of_death: Number(match[3]),
			place_of_death: match[4],
			exact_year_of_birth: false,
			exact_year_of_death: false
		};
	}

	match = birthData.match(
		ArtistLifePatters.CircaBirthYearKnownLocationKnownDeathYearUnknownLocation
	);
	if (match) {
		return {
			year_of_birth: Number(match[1]),
			place_of_birth: match[2],
			year_of_death: Number(match[3]),
			place_of_death: null,
			exact_year_of_birth: false,
			exact_year_of_death: true
		};
	}

	match = birthData.match(ArtistLifePatters.KnownToBeActiveInASingleLocation);
	if (match) {
		return {
			year_active_start: Number(match[1]),
			year_active_end: Number(match[2]),
			active_at_location: match[3]
		};
	}

	match = birthData.match(ArtistLifePatters.KnownToBeActiveAtInTimeNoLocation);
	if (match) {
		return {
			year_of_birth: Number(match[1]),
			year_of_death: Number(match[2]),
			exact_year_of_birth: true,
			exact_year_of_death: true
		};
	}

	match = birthData.match(ArtistLifePatters.KnwonDeathYearLocation);
	if (match) {
		return {
			year_of_death: Number(match[1]),
			place_of_death: match[2],
			exact_year_of_death: true
		};
	}

	match = birthData.match(ArtistLifePatters.KnownBirthYearRangeLocationKnownDeathYearRangeLocation);
	if (match) {
		const birthRangeStart = Number(match[1]);
		const birthRangeEnd = Number(match[2]);
		const deathRangeStart = Number(match[4]);
		const deathRangeEnd = Number(match[5]);
		return {
			year_of_birth: ArrayAverage([
				birthRangeStart,
				birthRangeEnd < 100 ? ExtrapolateSecondYear(birthRangeStart, birthRangeEnd) : birthRangeEnd
			]),
			year_of_death: ArrayAverage([
				deathRangeStart,
				deathRangeEnd < 100 ? ExtrapolateSecondYear(deathRangeStart, deathRangeEnd) : deathRangeEnd
			]),
			place_of_birth: match[3],
			place_of_death: match[4],
			exact_year_of_birth: false,
			exact_year_of_death: false
		};
	}

	match = birthData.match(
		ArtistLifePatters.KnownActiveYearLocationStartCircaDeathDeathYearKnownLocation
	);
	if (match) {
		return {
			year_active_start: Number(match[1]),
			active_at_location: match[2],
			year_of_death: Number(match[3]),
			place_of_death: match[4],
			exact_year_of_death: false
		};
	}

	match = birthData.match(ArtistLifePatters.KnownBirthYearLocationCircaDeathYearUnknownLocation);
	if (match) {
		return {
			year_of_birth: Number(match[1]),
			place_of_birth: match[2],
			year_of_death: Number(match[3]),
			place_of_death: null,
			exact_year_of_birth: true,
			exact_year_of_death: false
		};
	}

	return null;
};

/**
 * It reads in a JSON file of artists, generates a random 15 character string for each artist that
 * doesn't have an id, and then writes the updated JSON file back to the file system
 */
export const expandBioData = async () => {
	const artists: ArtistReferenceRecord[] = JSON.parse(
		fs.readFileSync('./reference/artists_with_bio.json', 'utf8')
	);

	let missingMetaInfo = 0;

	console.log(`Found ${artists.length} artists in the file.`);

	artists.map((artist) => {
		if (!artist.id) {
			/* PocketBase uses a 15 char random string as ID */
			//generate a 15 character random string with lowercase letters and numbers as id
			let id = makeid(15);

			/* Make sure the id is unique */
			while (idTracker.includes(id)) {
				id = makeid(15);
			}

			artist.id = id;
		}

		if (!artist.slug) {
			artist.slug = slugify(artist.ARTIST);
		}

		/* Get the wga_id from the artist.URL and the regex */
		const match = artist.URL.match(regex);
		if (match && !artist.wga_id) {
			artist.wga_id = match[1];
		}

		/* Get the meta info from the artist.BIRTH DATA */
		if (artist['BIRTH DATA']) {
			artist.meta = generateMetaInfo(artist['BIRTH DATA']);
			/* Check if the birth data matches any of the patterns */
		}

		if (!artist.meta) {
			console.table(artist);
			missingMetaInfo++;
		}
	});

	fs.writeFileSync('./reference/artists_with_bio.json', JSON.stringify(artists, null, 2));

	console.log(`Found ${missingMetaInfo} of ${artists.length} artists with missing meta info.`);
	console.log('Done!');
};

expandBioData();
