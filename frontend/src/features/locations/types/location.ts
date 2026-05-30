// Mirrors backend location service entities.

// Resolved room: full path returned by GET /locations/rooms/:id.
export interface ResolvedRoom {
  RoomID: string;
  RoomCode: string;
  RoomName: string;
  FloorID: string;
  FloorName: string;
  BuildingID: string;
  BuildingName: string;
}

export interface Building {
  ID: string;
  Name: string;
  Address: string;
  IsActive: boolean;
  CreatedAt: string;
  UpdatedAt: string;
}

export interface Floor {
  ID: string;
  BuildingID: string;
  Name: string;
  SortOrder: number;
}

export interface Room {
  ID: string;
  FloorID: string;
  Code: string;
  Name: string;
  IsActive: boolean;
}

export interface CustomerLocation {
  ID: string;
  CustomerID: string;
  RoomID: string;
  Label: string;
  IsDefault: boolean;
  CreatedAt: string;
  // Human-readable delivery path enriched by the location service List endpoint.
  BuildingName: string;
  FloorName: string;
  RoomCode: string;
  RoomName: string;
}
