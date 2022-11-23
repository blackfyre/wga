import * as fs from 'fs';
import * as crypto from 'crypto';

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
	exact_year_active_start?: boolean;
	exact_year_active_end?: boolean;
};

type ArtistReferenceRecord = {
	id: string;
	slug: string;
	name: string;
	wga_id: string;
	profession: string;
	school: string;
	meta: ArtistMetaInfo | null;
	source: ArtistRecordRaw;
	possibleInfluece: ArtPeriod[];
};

type ArtistRecordRaw = {
	ARTIST: string;
	'BIRTH DATA': string;
	PROFESSION: string;
	SCHOOL: string;
	URL: string;
};

type ArtPeriod = {
	id: string;
	name: string;
	start: number;
	end: number;
	description: string;
};

const idTracker: string[] = [];
let ArtPeriods: ArtPeriod[] = [];
const ArtistLifePatters = {
	KnownBirthYearLocationKnownDeathYearLocation:
		/^\(b\.?\s(\d{4})(?:\.|,)\s(.*),\s?(?:d\.?)?\s?(\d{4}),\s(.*)\)$/,
	CircaBornYearKnownLocationKnownDeathYearLocation:
		/^\(b\.?\sca\.?\s(\d{4}),\s(.*),\s*?d\.?\s(\d{4}),\s(.*)\)$/,
	CircaBornYearKnownLocationKnownDeathYearLocation2:
		/^\(b\.?\s(\d{4})s,\s(.*),\sd\.?\s(\d{4}),\s(.*)\)$/,
	CircaBornYearKnownLocationKnownDeathYearLocation3:
		/^\(b\.?\s(\d{4})\/(\d{2}),\s(.*),\sd\.?\s(\d{4}),\s(.*)\)$/,
	KnownBornYearLocationCircaDeathKnownLocation:
		/^\(b\.?\s(\d{4}),\s(.*),\sd\.?\s(?:ca\.?|after)\s?(\d{4}),\s(.*)\)$/,
	CircaBirthYearKnownLocationCircaDeathKnownLocation:
		/^\(b\.?\sca\.?\s(\d{4}),\s(.*),\sd\.?\s(?:ca\.?|after)\s*?(\d{4}),\s(.*)\)$/,
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
		/^\(b\.?\s(\d{4}),\s(.*),\sd\.?\s(?:ca\.?|after)\s?(\d{4})\)$/,
	ActiveYearRangeKnownLocation: /^\(active\s?ca?\.?\s?(\d{4})-(\d{4})\s?in\s?(.*)\)$/
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
	return Math.trunc(arr.reduce((p, c) => p + c, 0) / arr.length);
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
 * It takes a string and a number, and returns a string
 * @param {string} input - The string to hash.
 * @param {number} length - The length of the hash to generate.
 * @returns A hash of the input string.
 */
export function generateHash(input: string, length: number): string {
	const hash = crypto.createHash('md5').update(input).digest('hex');
	return hash.slice(0, length).toLowerCase();
}

/* Regex for the old ID */
const regex = /https:\/\/www\.wga\.hu\/bio\/[a-z]\/(.*)\/biograph\.html/;

/**
 * It takes a string of text that describes an artist's life, and returns an object that contains the
 * artist's birth year, birth location, death year, death location, and active years and locations
 * @param {string} birthData - The string that contains the artist's birth data.
 * @returns ArtistMetaInfo
 */
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

	match = birthData.match(ArtistLifePatters.CircaBornYearKnownLocationKnownDeathYearLocation3);
	if (match) {
		const birthRangeStart = Number(match[1]);
		const birthRangeEnd = Number(match[2]);
		return {
			year_of_birth: ArrayAverage([
				birthRangeStart,
				birthRangeEnd < 100 ? ExtrapolateSecondYear(birthRangeStart, birthRangeEnd) : birthRangeEnd
			]),
			place_of_birth: match[3],
			year_of_death: Number(match[4]),
			place_of_death: match[5],
			exact_year_of_birth: false,
			exact_year_of_death: true
		};
	}

	match = birthData.match(ArtistLifePatters.ActiveYearRangeKnownLocation);
	if (match) {
		return {
			year_active_start: Number(match[1]),
			year_active_end: Number(match[2]),
			active_at_location: match[3],
			exact_year_active_start: false,
			exact_year_active_end: false
		};
	}

	return null;
};

const LookUpArtPeriod = (year: number | null | undefined) => {
	const collector: ArtPeriod[] = [];

	if (!year || year === null) {
		return collector;
	}

	if (ArtPeriods.length === 0) {
		ArtPeriods = JSON.parse(fs.readFileSync('./reference/art_periods.json', 'utf8'));
	}

	ArtPeriods.forEach((period: ArtPeriod) => {
		if (year >= period.start && year <= period.end) {
			collector.push(period);
		}
	});

	return collector;
};

/**
 * It reads in a JSON file of artists, generates a random 15 character string for each artist that
 * doesn't have an id, and then writes the updated JSON file back to the file system
 */
export const expandBioData = async () => {
	const artists: ArtistRecordRaw[] = JSON.parse(
		fs.readFileSync('./reference/artists_with_bio_stage_0.json', 'utf8')
	);

	let missingMetaInfo = 0;
	const collector: ArtistReferenceRecord[] = [];

	console.log(`Found ${artists.length} artists in the file.`);

	artists.map((artist) => {
		const NewArtist = {
			name: artist.ARTIST,
			source: artist
		} as ArtistReferenceRecord;
		/* PocketBase uses a 15 char random string as ID */
		//generate a 15 character random string with lowercase letters and numbers as id
		let id = generateHash(artist.URL, 15);

		/* Make sure the id is unique */
		while (idTracker.includes(id)) {
			id = generateHash(JSON.stringify(artist), 15);
		}

		NewArtist.id = id;

		NewArtist.slug = slugify(NewArtist.name);

		/* Get the wga_id from the artist.URL and the regex */
		const match = artist.URL.match(regex);
		if (match) {
			NewArtist.wga_id = match[1];
		}

		NewArtist.meta = generateMetaInfo(artist['BIRTH DATA']);

		if (!NewArtist.meta) {
			console.warn(NewArtist);
			missingMetaInfo++;
		}

		NewArtist.possibleInfluece = [
			/* @ts-ignore */
			...LookUpArtPeriod(NewArtist?.meta?.year_of_birth | NewArtist?.meta?.year_active_start),
			/* @ts-ignore */
			...LookUpArtPeriod(NewArtist?.meta?.year_of_death | NewArtist?.meta?.year_active_end)
		];

		collector.push(NewArtist);
	});

	fs.writeFileSync('./reference/artists_with_bio_stage_1.json', JSON.stringify(collector, null, 2));

	console.log(`Found ${missingMetaInfo} of ${artists.length} artists with missing meta info.`);
	console.log('Done!');
};

expandBioData();
