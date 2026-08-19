package seed

import "github.com/Hlgxz/gai/database/orm"

// Seeder populates the database with initial data.
type Seeder func(db *orm.DB) error

// Run executes seeders in order.
func Run(db *orm.DB, seeders ...Seeder) error {
	for _, s := range seeders {
		if err := s(db); err != nil {
			return err
		}
	}
	return nil
}
