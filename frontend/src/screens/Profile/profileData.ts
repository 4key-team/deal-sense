import { getItem } from "../../lib/storage";

export interface CompanyProfileData {
  name: string;
  teamSize: string;
  experience: string;
  stack: string[];
  certs: string[];
  specializations: string[];
  clients: string;
  extra: string;
}

export const PROFILE_STORAGE_KEY = "company-profile";

export const EMPTY_PROFILE: CompanyProfileData = {
  name: "",
  teamSize: "",
  experience: "",
  stack: [],
  certs: [],
  specializations: [],
  clients: "",
  extra: "",
};

export function loadProfile(): CompanyProfileData {
  return getItem<CompanyProfileData>(PROFILE_STORAGE_KEY, EMPTY_PROFILE);
}
