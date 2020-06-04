import java.sql.Connection;
import java.sql.DriverManager;
import java.sql.ResultSet;
import java.sql.Statement;
import java.util.ArrayList;
import java.io.*;
import java.util.Scanner;
import java.util.List;
    
public class InQueryFile {
   static List<String> buff = new ArrayList<String>();  
   static Connection c = null;
   static Statement stmt = null;
   static int maxBuffLen = 50;
   
   public static String quote(String astr) {
       //TODO: Add approapriate escaping here
       return "'" + astr + "'";
   }
   
   public static void flush() {
     if (buff.size() < 1) {
         return;
     }
     try {        
        String oidsStr =  String.join(", ", buff);
        String squery = "SELECT DISTINCT paroid, partbl FROM omap WHERE omap.chiloid IN (" + oidsStr + " );";
        long startSQL = System.nanoTime();    
        ResultSet rs = stmt.executeQuery( squery );
        long finishSQL = System.nanoTime();    
        while ( rs.next() ) {
          String  partbl = rs.getString("partbl");
          String  paroid = rs.getString("paroid");
          System.out.println( "partbl = " + partbl + " paroid=" + paroid );
        }
        rs.close();
        long finishSQLIter = System.nanoTime();    
        System.out.println( "sqlexec=" + Double.toString((finishSQL - startSQL)*0.000001) + " sqlIter=" + Double.toString((finishSQLIter - finishSQL)*0.000001));
     } catch ( Exception e ) {
         System.err.println( e.getClass().getName()+": "+ e.getMessage() );
         System.exit(0);
     }
     buff.clear(); // clear buffer for next function 
   }
   
   public static void main( String args[] ) {
      try {
         // TODO: Get File Name for input file from ARGS
         Class.forName("org.postgresql.Driver");
         c = DriverManager
            .getConnection("jdbc:postgresql://localhost:5432/oidmap", "postgres", "test");
         c.setAutoCommit(false);
         System.out.println("Opened database successfully");
         stmt = c.createStatement();
         FileInputStream fis=new FileInputStream("../test.map.txt");
         Scanner sc=new Scanner(fis); 
         sc.nextLine(); // skip header line
         while(sc.hasNextLine())  
         {
            String tline = sc.nextLine();
            //System.out.println(tline);      //returns the line that was skipped  
            String[] fields = tline.split(",");
            if (fields.length == 4) {
              buff.add(quote(fields[3]));
              if (buff.size() > maxBuffLen) {
                  flush();
              }
            }
         }
         flush();
         stmt.close();
         c.close();
         sc.close();  //closes the scanner  
      } catch ( Exception e ) {
         System.err.println( e.getClass().getName()+": "+ e.getMessage() );
         System.exit(0);
      }
      System.out.println("Operation done successfully");
   }
}